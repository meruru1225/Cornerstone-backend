package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
)

// MaxOffsetLimit Elastic 深分页限制
const MaxOffsetLimit = 10000

type PostService interface {
	RecommendPost(ctx context.Context, sessionID string, cursor string, pageSize int) (*dto.PostWaterfallDTO, error)
	SearchPost(ctx context.Context, keyword string, page, pageSize int) (*dto.PostWaterfallDTO, error)
	SearchPostMe(ctx context.Context, userID uint64, keyword string, page, pageSize int) (*dto.PostWaterfallDTO, error)
	LastestPost(ctx context.Context, page, pageSize int) (*dto.PostWaterfallDTO, error)
	CreatePost(ctx context.Context, userID uint64, postDTO *dto.PostBaseDTO) error
	GetPostById(ctx context.Context, postID uint64) (*dto.PostDTO, error)
	GetPost(ctx context.Context, userID uint64, postID uint64) (*dto.PostDTO, error)
	GetPostByIds(ctx context.Context, ids []uint64) ([]*dto.PostDTO, error)
	GetPostByUserId(ctx context.Context, userId uint64, page, pageSize int) (*dto.PostWaterfallDTO, error)
	GetPostSelf(ctx context.Context, userId uint64, page, pageSize int) (*dto.PostWaterfallDTO, error)
	GetPostByTag(ctx context.Context, tag string, isMain bool, page, pageSize int) (*dto.PostWaterfallDTO, error)
	GetWarningPosts(ctx context.Context, lastID uint64, pageSize int) (*dto.PostWaterfallDTO, error)
	UpdatePostStatus(ctx context.Context, postID uint64, status int) error
	UpdatePostContent(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error
	UpdatePostCounts(ctx context.Context, pid uint64, likes int64, comments int64, collects int64, views int64) error
	DeletePost(ctx context.Context, userID uint64, postID uint64) error
}

type postServiceImpl struct {
	postESRepo       es.PostRepo
	postDBRepo       repository.PostRepo
	userInterestRepo repository.UserInterestRepo
}

func NewPostService(postESRepo es.PostRepo, postDBRepo repository.PostRepo, userInterestRepo repository.UserInterestRepo) PostService {
	return &postServiceImpl{
		postESRepo:       postESRepo,
		postDBRepo:       postDBRepo,
		userInterestRepo: userInterestRepo,
	}
}

// RecommendPost 推荐流
func (s *postServiceImpl) RecommendPost(ctx context.Context, sessionID string, cursor string, pageSize int) (*dto.PostWaterfallDTO, error) {
	userID := ctx.Value("user_id").(uint64)

	key := consts.UserInterestKey + strconv.FormatUint(userID, 10)
	tags, _ := redis.ZRevRange(ctx, key, 0, 9)

	// 冷启动
	if len(tags) == 0 && userID > 0 {
		userIDStr := strconv.FormatUint(userID, 10)
		lockKey := consts.UserInterestInitLock + userIDStr
		lockUUID := uuid.NewString()

		ok, err := redis.TryLock(ctx, lockKey, lockUUID, 5*time.Second, 0)
		if err == nil && ok {
			defer redis.UnLock(ctx, lockKey, lockUUID)
			tags, _ = redis.ZRevRange(ctx, key, 0, 9)
			if len(tags) == 0 {
				snapshot, err := s.userInterestRepo.GetUserInterests(ctx, userID)
				if err == nil && snapshot != nil && len(snapshot.Interests) > 0 {
					type kv struct {
						Key   string
						Value int64
					}
					var ss []kv
					for k, v := range snapshot.Interests {
						ss = append(ss, kv{k, v})
					}
					sort.Slice(ss, func(i, j int) bool {
						return ss[i].Value > ss[j].Value
					})
					for _, item := range ss {
						_ = redis.ZAdd(ctx, key, float64(item.Value), item.Key)
					}
					_ = redis.ZRemRangeByRank(ctx, key, 0, -101)
					_ = redis.Expire(ctx, key, 24*time.Hour)
					tags, _ = redis.ZRevRange(ctx, key, 0, 9)
				}
			}
		}
	}

	lastSortValues, err := util.DecodeCursor(cursor)
	if err != nil {
		log.ErrorContext(ctx, "decode cursor error", "err", err)
		lastSortValues = nil
	}
	// 生成随机种子
	seed := util.HashSessionID(sessionID)
	var vector []float32
	var interestText string
	if len(tags) > 0 {
		interestText = strings.Join(tags, " ")
		timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()

		v, vErr := llm.GetVectorByString(timeoutCtx, interestText)
		if vErr != nil {
			log.WarnContext(ctx, "llm get vector failed or timeout", "err", vErr)
		} else {
			vector = v
		}
	}

	fetchSize := pageSize * 2
	var candidates []*es.PostES
	isFallbackMode := false
	if len(lastSortValues) == 1 {
		isFallbackMode = true
	}
	if !isFallbackMode {
		candidates, err = s.postESRepo.RecommendPosts(ctx, interestText, vector, lastSortValues, fetchSize, seed)
		if err != nil {
			candidates = []*es.PostES{}
		}
	}

	targetSize := pageSize + 1
	finalPosts := make([]*es.PostES, 0, targetSize)
	addedMap := make(map[uint64]struct{})
	viewedKey := consts.UserViewedKey + strconv.FormatUint(userID, 10)
	rdb := redis.GetRdbClient()

	for _, post := range candidates {
		isViewed, _ := rdb.SIsMember(ctx, viewedKey, post.ID).Result()
		if isViewed {
			continue
		}

		finalPosts = append(finalPosts, post)
		addedMap[post.ID] = struct{}{}

		if len(finalPosts) >= targetSize {
			break
		}
	}

	// 降级填充
	if len(finalPosts) < targetSize {
		needed := targetSize - len(finalPosts)
		var latestPosts []*es.PostES
		var err error

		if isFallbackMode {
			latestPosts, err = s.postESRepo.GetLatestPostsByCursor(ctx, lastSortValues, needed*2)
		} else {
			latestPosts, err = s.postESRepo.GetLatestPosts(ctx, 0, needed*3)
		}

		if err == nil {
			for _, p := range latestPosts {
				if _, ok := addedMap[p.ID]; ok {
					continue
				}
				isViewed, _ := rdb.SIsMember(ctx, viewedKey, p.ID).Result()
				if isViewed {
					continue
				}

				finalPosts = append(finalPosts, p)
				addedMap[p.ID] = struct{}{}
				if len(finalPosts) >= targetSize {
					break
				}
			}
		}
	}

	hasMore := false
	if len(finalPosts) > pageSize {
		hasMore = true
		finalPosts = finalPosts[:pageSize]
	}

	// 记录已读
	if len(finalPosts) > 0 && userID > 0 {
		go func(ids []*es.PostES) {
			pipe := redis.GetRdbClient().Pipeline()
			bgCtx := context.Background()

			for _, p := range ids {
				pipe.SAdd(bgCtx, viewedKey, p.ID)
			}
			pipe.Expire(bgCtx, viewedKey, 72*time.Hour)
			_, _ = pipe.Exec(bgCtx)
		}(finalPosts)
	}

	dtoItems, err := s.batchToPostDTOByES(finalPosts)
	if err != nil {
		return nil, err
	}

	// 计算 Next Cursor
	var nextCursor string
	if len(finalPosts) > 0 {
		lastPost := finalPosts[len(finalPosts)-1]
		if len(lastPost.Sort) > 0 {
			nextCursor = util.EncodeCursor(lastPost.Sort)
		}
	}

	return &dto.PostWaterfallDTO{
		List:       dtoItems,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// SearchPost 搜索流
func (s *postServiceImpl) SearchPost(ctx context.Context, keyword string, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	if (page-1)*pageSize >= MaxOffsetLimit {
		return &dto.PostWaterfallDTO{
			List:       []*dto.PostDTO{},
			HasMore:    false,
			NextCursor: "",
		}, nil
	}

	vector, err := llm.GetVectorByString(ctx, keyword)
	if err != nil {
		return nil, err
	}

	from := (page - 1) * pageSize

	return getWaterfallPosts(pageSize,
		func() ([]*es.PostES, error) {
			return s.postESRepo.HybridSearch(ctx, keyword, vector, from, pageSize+1)
		},
		s.batchToPostDTOByES,
	)
}

func (s *postServiceImpl) SearchPostMe(ctx context.Context, userID uint64, keyword string, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	if (page-1)*pageSize >= MaxOffsetLimit {
		return &dto.PostWaterfallDTO{
			List:    []*dto.PostDTO{},
			HasMore: false,
		}, nil
	}

	vector, err := llm.GetVectorByString(ctx, keyword)
	if err != nil {
		return nil, err
	}

	from := (page - 1) * pageSize

	return getWaterfallPosts(pageSize,
		func() ([]*es.PostES, error) {
			return s.postESRepo.HybridSearchMe(ctx, userID, keyword, vector, from, pageSize+1)
		},
		s.batchToPostDTOByES,
	)
}

// LastestPost 最新流
func (s *postServiceImpl) LastestPost(ctx context.Context, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	if (page-1)*pageSize >= MaxOffsetLimit {
		return &dto.PostWaterfallDTO{
			List:       []*dto.PostDTO{},
			HasMore:    false,
			NextCursor: "",
		}, nil
	}

	from := (page - 1) * pageSize

	return getWaterfallPosts(pageSize,
		func() ([]*es.PostES, error) {
			return s.postESRepo.GetLatestPosts(ctx, from, pageSize+1)
		},
		s.batchToPostDTOByES,
	)
}

// CreatePost 创建帖子
func (s *postServiceImpl) CreatePost(ctx context.Context, userID uint64, postDTO *dto.PostBaseDTO) error {
	var hdelKeys []string

	for _, mediaDTO := range postDTO.Medias {
		if err := processMedia(ctx, mediaDTO, &hdelKeys); err != nil {
			return err
		}
	}

	post := &model.Post{}
	if err := copier.Copy(post, postDTO); err != nil {
		return err
	}
	if err := copier.Copy(&post.MediaList, &postDTO.Medias); err != nil {
		return err
	}
	post.UserID = userID

	if err := s.postDBRepo.CreatePost(ctx, post); err != nil {
		return err
	}

	if len(hdelKeys) > 0 {
		go func(keys []string) {
			_ = redis.HDel(context.Background(), consts.MediaTempKey, hdelKeys...)
		}(hdelKeys)
	}

	return nil
}

// GetPostById 获取单个帖子
func (s *postServiceImpl) GetPostById(ctx context.Context, postID uint64) (*dto.PostDTO, error) {
	post, err := s.postDBRepo.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}
	return s.toPostDTO(post)
}

// GetPost 获取单个帖子
func (s *postServiceImpl) GetPost(ctx context.Context, userID uint64, PostID uint64) (*dto.PostDTO, error) {
	post, err := s.postESRepo.GetPostById(ctx, PostID)
	if err != nil {
		return nil, err
	}
	if post == nil ||
		(post.UserID != userID && post.Status != consts.PostStatusNormal) {
		return nil, ErrPostNotFound
	}

	go func(uid uint64, tags []string) {
		s.RecordInterest(context.Background(), uid, tags, 1)
	}(userID, post.AITags)

	return s.toPostDTOByES(post)
}

// GetPostByIds 批量获取帖子
func (s *postServiceImpl) GetPostByIds(ctx context.Context, ids []uint64) ([]*dto.PostDTO, error) {
	posts, err := s.postDBRepo.GetPostByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	return s.batchToPostDTO(posts)
}

func (s *postServiceImpl) GetPostByUserId(ctx context.Context, userId uint64, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	return getWaterfallPosts(pageSize,
		func() ([]*model.Post, error) {
			return s.postDBRepo.GetPostByUserId(ctx, userId, pageSize, (page-1)*pageSize)
		},
		s.batchToPostDTO,
	)
}

// GetPostSelf 获取登录用户自己的帖子列表
func (s *postServiceImpl) GetPostSelf(ctx context.Context, userId uint64, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	return getWaterfallPosts(pageSize,
		func() ([]*model.Post, error) {
			return s.postDBRepo.GetPostSelf(ctx, userId, pageSize+1, (page-1)*pageSize)
		},
		func(posts []*model.Post) ([]*dto.PostDTO, error) {
			out := make([]*dto.PostDTO, len(posts))
			for i, post := range posts {
				item, err := s.toPostDTO(post)
				if err != nil {
					return nil, err
				}
				item.Status = &post.Status
				out[i] = item
			}
			return out, nil
		},
	)
}

// GetPostByTag 根据标签获取帖子
func (s *postServiceImpl) GetPostByTag(ctx context.Context, tag string, isMain bool, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	return getWaterfallPosts(pageSize,
		func() ([]*es.PostES, error) {
			return s.postESRepo.GetPostByTag(ctx, tag, isMain, page-1, pageSize)
		},
		s.batchToPostDTOByES,
	)
}

// GetWarningPosts 获取所有待审核/警告状态的帖子
func (s *postServiceImpl) GetWarningPosts(ctx context.Context, lastID uint64, pageSize int) (*dto.PostWaterfallDTO, error) {
	// 1. 调用 Repo 使用游标查询
	rawData, err := s.postDBRepo.GetPostsByStatusCursor(ctx, 2, lastID, pageSize+1)
	if err != nil {
		return nil, err
	}

	// 2. 判定 HasMore
	hasMore := false
	if len(rawData) > pageSize {
		hasMore = true
		rawData = rawData[:pageSize]
	}

	// 3. 转换 DTO
	dtoItems, err := s.batchToPostDTO(rawData)
	if err != nil {
		return nil, err
	}

	return &dto.PostWaterfallDTO{
		List:    dtoItems,
		HasMore: hasMore,
	}, nil
}

// UpdatePostStatus 更改帖子状态
func (s *postServiceImpl) UpdatePostStatus(ctx context.Context, postID uint64, status int) error {
	err := s.postDBRepo.UpdatePostStatus(ctx, postID, status)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePostContent 更新帖子内容及媒体
func (s *postServiceImpl) UpdatePostContent(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error {
	oldPost, err := s.postDBRepo.GetPostByAllStatus(ctx, postID)
	if err != nil {
		return err
	}
	if oldPost == nil {
		return ErrPostNotFound
	}
	if oldPost.UserID != userID {
		return UnauthorizedError
	}

	newMediaMap := make(map[string]struct{})
	for _, m := range postDTO.Medias {
		newMediaMap[m.MediaURL] = struct{}{}
	}

	var toDeleteKeys []string
	for _, oldMedia := range oldPost.MediaList {
		if _, exists := newMediaMap[oldMedia.MediaURL]; !exists {
			toDeleteKeys = append(toDeleteKeys, oldMedia.MediaURL)
			if oldMedia.CoverURL != nil && *oldMedia.CoverURL != "" {
				toDeleteKeys = append(toDeleteKeys, *oldMedia.CoverURL)
			}
		}
	}

	var hdelKeys []string
	for _, mediaDTO := range postDTO.Medias {
		isAlreadyInOld := false
		for _, oldMedia := range oldPost.MediaList {
			if oldMedia.MediaURL == mediaDTO.MediaURL {
				isAlreadyInOld = true
				break
			}
		}

		if !isAlreadyInOld {
			if err = processMedia(ctx, mediaDTO, &hdelKeys); err != nil {
				return err
			}
		}
	}

	if err = copier.Copy(oldPost, postDTO); err != nil {
		return err
	}
	var newMediaList model.MediaList
	if err = copier.Copy(&newMediaList, &postDTO.Medias); err != nil {
		return err
	}
	oldPost.MediaList = newMediaList

	if err = s.postDBRepo.UpdatePostContent(ctx, oldPost); err != nil {
		return err
	}

	go func() {
		bgCtx := context.Background()
		for _, key := range toDeleteKeys {
			_ = minio.DeleteFile(bgCtx, key)
		}
		if len(hdelKeys) > 0 {
			_ = redis.HDel(bgCtx, consts.MediaTempKey, hdelKeys...)
		}
	}()

	return nil
}

// UpdatePostCounts 更新帖子计数
func (s *postServiceImpl) UpdatePostCounts(ctx context.Context, pid uint64, likes int64, comments int64, collects int64, views int64) error {
	return s.postDBRepo.UpdatePostCounts(ctx, pid, likes, comments, collects, views)
}

// DeletePost 删除帖子
func (s *postServiceImpl) DeletePost(ctx context.Context, userID uint64, postID uint64) error {
	post, err := s.postDBRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}
	if post.UserID != userID {
		return UnauthorizedError
	}

	if err = s.postDBRepo.DeletePost(ctx, postID); err != nil {
		return err
	}

	if len(post.MediaList) > 0 {
		go func() {
			for _, m := range post.MediaList {
				_ = minio.DeleteFile(context.Background(), m.MediaURL)
			}
		}()
	}

	return nil
}

func (s *postServiceImpl) RecordInterest(ctx context.Context, userID uint64, aiTags []string, actionType int) {
	if userID == 0 || len(aiTags) == 0 {
		return
	}

	var tagsToAdd []string
	if actionType == 1 {
		limit := 2
		if len(aiTags) < 2 {
			limit = len(aiTags)
		}
		tagsToAdd = aiTags[:limit]
	} else {
		tagsToAdd = aiTags
	}

	now := time.Now().Unix()
	userIDStr := strconv.FormatUint(userID, 10)
	key := consts.UserInterestKey + userIDStr
	exists, _ := redis.Exists(ctx, key)
	if !exists {
		newUUID, err := uuid.NewUUID()
		if err != nil {
			return
		}
		lockKey := consts.UserInterestInitLock + userIDStr
		ok, err := redis.TryLock(ctx, lockKey, newUUID.String(), 5, 0)
		if err != nil || !ok {
			return
		}
		defer redis.UnLock(ctx, lockKey, newUUID.String())
		if ex, _ := redis.Exists(ctx, key); !ex {
			snapshot, err := s.userInterestRepo.GetUserInterests(ctx, userID)
			if err == nil && snapshot != nil && len(snapshot.Interests) > 0 {
				for tag, score := range snapshot.Interests {
					_ = redis.ZAdd(ctx, key, float64(score), tag)
				}
				_ = redis.Expire(ctx, key, 24*time.Hour)
			}
		}
	}

	for _, tag := range tagsToAdd {
		_ = redis.ZAdd(ctx, key, float64(now), tag)
	}

	_ = redis.Expire(ctx, key, 24*time.Hour)
	_ = redis.ZRemRangeByRank(ctx, key, 0, -101)

	_ = redis.SAdd(ctx, consts.UserInterestDirtyKey, userID)
}

// toPostDTO 将 Model 转换为返回给前端的 DTO
func (s *postServiceImpl) toPostDTO(post *model.Post) (*dto.PostDTO, error) {
	out := &dto.PostDTO{}
	if err := copier.Copy(out, post); err != nil {
		return nil, err
	}
	if err := copier.Copy(&out.Medias, &post.MediaList); err != nil {
		return nil, err
	}
	for _, m := range out.Medias {
		m.MediaURL = minio.GetPublicURL(m.MediaURL)
		if m.CoverURL != nil {
			url := minio.GetPublicURL(*m.CoverURL)
			m.CoverURL = &url
		}
	}

	defaultAvatarUrl := minio.GetPublicURL("default_avatar.png")

	if post.User.ID > 0 {
		out.UserID = post.User.ID
		if post.User.UserDetail.UserID > 0 {
			out.Nickname = post.User.UserDetail.Nickname
			out.AvatarURL = minio.GetPublicURL(post.User.UserDetail.AvatarURL)
		} else {
			out.Nickname = "用户_" + strconv.FormatUint(post.User.ID, 10)
			out.AvatarURL = defaultAvatarUrl
		}
	} else {
		out.Nickname = "未知用户"
		out.AvatarURL = defaultAvatarUrl
	}

	return out, nil
}

func (s *postServiceImpl) toPostDTOByES(post *es.PostES) (*dto.PostDTO, error) {
	out := &dto.PostDTO{}
	if err := copier.Copy(out, post); err != nil {
		return nil, err
	}
	out.Nickname = post.UserNickname
	out.AvatarURL = minio.GetPublicURL(post.UserAvatar)
	out.CreatedAt = post.CreatedAt.Format("2006-01-02 15:04:05")
	out.UpdatedAt = post.UpdatedAt.Format("2006-01-02 15:04:05")
	var mediaBaseDTO []*dto.MediasBaseDTO
	for _, media := range post.Media {
		var url string
		if media.Cover != nil && *media.Cover != "" {
			url = minio.GetPublicURL(*media.Cover)
		}
		mediaBaseDTO = append(mediaBaseDTO, &dto.MediasBaseDTO{
			MimeType: media.Type,
			MediaURL: minio.GetPublicURL(media.URL),
			Width:    media.Width,
			Height:   media.Height,
			Duration: media.Duration,
			CoverURL: &url,
		})
	}
	out.Medias = mediaBaseDTO

	return out, nil
}

// batchToPostDTO 批量转换辅助
func (s *postServiceImpl) batchToPostDTO(posts []*model.Post) ([]*dto.PostDTO, error) {
	out := make([]*dto.PostDTO, len(posts))
	for i, post := range posts {
		item, err := s.toPostDTO(post)
		if err != nil {
			return nil, err
		}
		out[i] = item
	}
	return out, nil
}

func (s *postServiceImpl) batchToPostDTOByES(posts []*es.PostES) ([]*dto.PostDTO, error) {
	out := make([]*dto.PostDTO, len(posts))
	for i, post := range posts {
		item, err := s.toPostDTOByES(post)
		if err != nil {
			return nil, err
		}
		out[i] = item
	}
	return out, nil
}

func getCover(ctx context.Context, mediaDTO *dto.MediasBaseDTO) error {
	if strings.HasPrefix(mediaDTO.MimeType, "video") &&
		mediaDTO.CoverURL == nil {
		stream, err := util.GetCover(ctx, minio.GetPublicURL(mediaDTO.MediaURL))
		if err != nil {
			return err
		}
		coverName := time.Now().Format("2006/01/02/") + uuid.NewString() + ".jpg"
		var fileKey string
		if seeker, ok := stream.(interface{ Size() int64 }); ok {
			fileKey, err = minio.UploadFile(ctx, coverName, stream, seeker.Size(), "image/jpeg")
		} else {
			fileKey, err = minio.UploadFile(ctx, coverName, stream, -1, "image/jpeg")
		}
		if err != nil {
			return err
		}
		mediaDTO.CoverURL = &fileKey
	}
	return nil
}

func verifyAndFillMediaMeta(ctx context.Context, mediaDTO *dto.MediasBaseDTO) error {
	val, err := redis.HGet(ctx, consts.MediaTempKey, mediaDTO.MediaURL)
	if err != nil || val == "" {
		log.WarnContext(ctx, "media resource not found in temp cache", "url", mediaDTO.MediaURL)
		return ErrFileNotExist
	}

	var meta dto.MediaTempMetadata
	if err = json.Unmarshal([]byte(val), &meta); err != nil {
		log.ErrorContext(ctx, "unmarshal media meta failed", "url", mediaDTO.MediaURL, "err", err)
		return UnExpectedError
	}

	mediaDTO.Width = meta.Width
	mediaDTO.Height = meta.Height
	mediaDTO.Duration = meta.Duration
	mediaDTO.MimeType = meta.MimeType

	return nil
}

func processMedia(ctx context.Context, mediaDTO *dto.MediasBaseDTO, hdelKeys *[]string) error {
	if err := verifyAndFillMediaMeta(ctx, mediaDTO); err != nil {
		return err
	}
	if err := getCover(ctx, mediaDTO); err != nil {
		return err
	}
	*hdelKeys = append(*hdelKeys, mediaDTO.MediaURL)
	if mediaDTO.CoverURL != nil && *mediaDTO.CoverURL != "" {
		*hdelKeys = append(*hdelKeys, *mediaDTO.CoverURL)
	}
	return nil
}

func getWaterfallPosts[T any](
	pageSize int,
	fetchFunc func() ([]T, error),
	convertFunc func([]T) ([]*dto.PostDTO, error),
) (*dto.PostWaterfallDTO, error) {
	rawData, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	hasMore := false
	if len(rawData) > pageSize {
		hasMore = true
		rawData = rawData[:pageSize]
	}

	dtoItems, err := convertFunc(rawData)
	if err != nil {
		return nil, err
	}

	return &dto.PostWaterfallDTO{
		List:    dtoItems,
		HasMore: hasMore,
	}, nil
}
