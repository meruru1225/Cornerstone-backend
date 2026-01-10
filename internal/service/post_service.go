package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/copier"
)

type PostService interface {
	RecommendPost(ctx context.Context, userID uint64, page, pageSize int) (*dto.PostWaterfallDTO, error)
	SearchPost(ctx context.Context, keyword string, page, pageSize int) (*dto.PostWaterfallDTO, error)
	CreatePost(ctx context.Context, userID uint64, postDTO *dto.PostBaseDTO) error
	GetPostById(ctx context.Context, postID uint64) (*dto.PostDTO, error)
	GetPost(ctx context.Context, userID uint64, postID uint64) (*dto.PostDTO, error)
	GetPostByIds(ctx context.Context, ids []uint64) ([]*dto.PostDTO, error)
	GetPostByUserId(ctx context.Context, userId uint64, page, pageSize int) (*dto.PostWaterfallDTO, error)
	GetPostSelf(ctx context.Context, userId uint64, page, pageSize int) (*dto.PostWaterfallDTO, error)
	UpdatePostContent(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error
	UpdatePostCounts(ctx context.Context, pid uint64, likes int64, comments int64, collects int64, views int64) error
	DeletePost(ctx context.Context, userID uint64, postID uint64) error
}

type postServiceImpl struct {
	postESRepo es.PostRepo
	postDBRepo repository.PostRepo
}

func NewPostService(postESRepo es.PostRepo, postDBRepo repository.PostRepo) PostService {
	return &postServiceImpl{
		postESRepo: postESRepo,
		postDBRepo: postDBRepo,
	}
}

func (s *postServiceImpl) RecommendPost(ctx context.Context, userID uint64, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	key := consts.UserInterestKey + strconv.FormatUint(userID, 10)
	from := (page - 1) * pageSize

	tags, err := redis.ZRevRange(ctx, key, 0, 9)
	if err != nil {
		log.ErrorContext(ctx, "get user interests from redis error", "err", err, "userID", userID)
	}

	if len(tags) > 0 {
		interestText := strings.Join(tags, " ")

		vector, err := llm.GetVectorByString(ctx, interestText)
		if err != nil {
			return nil, err
		}

		return getWaterfallPosts(pageSize,
			func() ([]*es.PostES, error) {
				return s.postESRepo.HybridSearch(ctx, interestText, vector, from, pageSize+1)
			},
			s.batchToPostDTOByES,
		)
	}

	return getWaterfallPosts(pageSize,
		func() ([]*es.PostES, error) {
			return s.postESRepo.GetLatestPosts(ctx, from, pageSize+1)
		},
		s.batchToPostDTOByES,
	)
}

// SearchPost 搜索帖子
func (s *postServiceImpl) SearchPost(ctx context.Context, keyword string, page, pageSize int) (*dto.PostWaterfallDTO, error) {
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

// CreatePost 创建帖子
func (s *postServiceImpl) CreatePost(ctx context.Context, userID uint64, postDTO *dto.PostBaseDTO) error {
	post := &model.Post{}
	if err := copier.Copy(post, postDTO); err != nil {
		return err
	}
	if err := copier.Copy(&post.MediaList, &postDTO.Medias); err != nil {
		return err
	}

	post.UserID = userID

	return s.postDBRepo.CreatePost(ctx, post)
}

// GetPostById 获取单个帖子
func (s *postServiceImpl) GetPostById(ctx context.Context, postID uint64) (*dto.PostDTO, error) {
	post, err := s.postDBRepo.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	return s.toPostDTO(post)
}

// GetPost 获取单个帖子
func (s *postServiceImpl) GetPost(ctx context.Context, userID uint64, PostID uint64) (*dto.PostDTO, error) {
	post, err := s.postESRepo.GetPostById(ctx, PostID)
	if err != nil {
		return nil, err
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
			return s.postDBRepo.GetPostByUserId(ctx, userId, pageSize+1, (page-1)*pageSize)
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

// UpdatePostContent 更新帖子内容及媒体
func (s *postServiceImpl) UpdatePostContent(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error {
	oldPost, err := s.postDBRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}
	if oldPost.UserID != userID {
		return UnauthorizedError
	}

	if err = copier.Copy(oldPost, postDTO); err != nil {
		return err
	}
	if err = copier.Copy(&oldPost.MediaList, &postDTO.Medias); err != nil {
		return err
	}

	return s.postDBRepo.UpdatePostContent(ctx, oldPost)
}

// UpdatePostCounts 更新帖子计数
func (s *postServiceImpl) UpdatePostCounts(ctx context.Context, pid uint64, likes int64, comments int64, collects int64, views int64) error {
	return s.postDBRepo.UpdatePostCounts(ctx, pid, likes, comments, collects, views)
}

// DeletePost 删除帖子
func (s *postServiceImpl) DeletePost(ctx context.Context, userID uint64, postID uint64) error {
	// 1. 鉴权
	post, err := s.postDBRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}
	if post.UserID != userID {
		return UnauthorizedError
	}
	return s.postDBRepo.DeletePost(ctx, postID)
}

func (s *postServiceImpl) RecordInterest(ctx context.Context, userID uint64, aiTags []string, actionType int) {
	if len(aiTags) == 0 {
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
	key := consts.UserInterestKey + strconv.FormatUint(userID, 10)

	for _, tag := range tagsToAdd {
		_ = redis.ZAdd(ctx, key, float64(now), tag)
	}

	_ = redis.ZRemRangeByRank(ctx, key, 0, -101)
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

	if post.User.ID > 0 {
		out.UserID = post.User.ID
		if post.User.UserDetail.UserID > 0 {
			out.Nickname = post.User.UserDetail.Nickname
			out.AvatarURL = post.User.UserDetail.AvatarURL
		} else {
			out.Nickname = "用户_" + strconv.FormatUint(post.User.ID, 10)
			out.AvatarURL = "default_avatar.png"
		}
	} else {
		out.Nickname = "未知用户"
		out.AvatarURL = "default_avatar.png"
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
		mediaBaseDTO = append(mediaBaseDTO, &dto.MediasBaseDTO{
			MimeType: media.Type,
			MediaURL: media.URL,
			Width:    media.Width,
			Height:   media.Height,
			Duration: media.Duration,
			CoverURL: media.Cover,
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
