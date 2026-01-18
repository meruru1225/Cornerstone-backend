package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	"errors"
	log "log/slog"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/copier"
	redisv9 "github.com/redis/go-redis/v9"
)

const (
	CommentStatusPending  int8 = 0
	CommentStatusApproved int8 = 1
)

type PostActionService interface {
	LikePost(ctx context.Context, userID, postID uint64) error
	CancelLikePost(ctx context.Context, userID, postID uint64) error
	GetPostLikeCount(ctx context.Context, postID uint64) (int64, error)
	GetPostLikeCounts(ctx context.Context, postIDs []uint64) ([]int64, error)
	IsLiked(ctx context.Context, userID, postID uint64) (bool, error)
	GetLikedPosts(ctx context.Context, userID uint64, page, pageSize int) (*dto.PostWaterfallDTO, error)

	CollectPost(ctx context.Context, userID, postID uint64) error
	CancelCollectPost(ctx context.Context, userID, postID uint64) error
	GetPostCollectionCount(ctx context.Context, postID uint64) (int64, error)
	IsCollected(ctx context.Context, userID, postID uint64) (bool, error)
	GetCollectedPosts(ctx context.Context, userID uint64, page, pageSize int) (*dto.PostWaterfallDTO, error)

	CreateComment(ctx context.Context, userID uint64, req *dto.CommentCreateDTO) error
	DeleteComment(ctx context.Context, userID, commentID uint64) error
	GetPostCommentCount(ctx context.Context, postID uint64) (int64, error)
	GetCommentsByPostID(ctx context.Context, postID uint64, page, pageSize int) ([]*dto.CommentDTO, error)
	GetSubComments(ctx context.Context, rootID uint64, page, pageSize int) ([]*dto.CommentDTO, error)

	LikeComment(ctx context.Context, userID, commentID uint64) error
	CancelLikeComment(ctx context.Context, userID, commentID uint64) error
	GetCommentLikeCount(ctx context.Context, commentID uint64) (int64, error)
	SyncCommentLikesCount(ctx context.Context, commentID uint64, count int) error

	TrackPostView(ctx context.Context, userID, postID uint64) error
	GetPostViewCount(ctx context.Context, postID uint64) (int64, error)

	ReportPost(ctx context.Context, userID, postID uint64) error
}

type postActionServiceImpl struct {
	actionRepo repository.PostActionRepo
	postRepo   repository.PostRepo
	userRepo   repository.UserRepo
}

const cacheExpiration = 7 * 24 * time.Hour

func NewPostActionService(
	actionRepo repository.PostActionRepo,
	postRepo repository.PostRepo,
	userRepo repository.UserRepo,
) PostActionService {
	return &postActionServiceImpl{
		actionRepo: actionRepo,
		postRepo:   postRepo,
		userRepo:   userRepo,
	}
}

func (s *postActionServiceImpl) LikePost(ctx context.Context, userID, postID uint64) error {
	return s.performAction(s.getPostCheck(ctx, postID), func() error {
		return s.actionRepo.CreateLike(ctx, &model.Like{UserID: userID, PostID: postID, CreatedAt: time.Now()})
	})
}

func (s *postActionServiceImpl) CancelLikePost(ctx context.Context, userID, postID uint64) error {
	return s.revokeAction(s.getPostCheck(ctx, postID), func() error {
		return s.actionRepo.DeleteLike(ctx, userID, postID)
	})
}

func (s *postActionServiceImpl) GetPostLikeCount(ctx context.Context, postID uint64) (int64, error) {
	key := consts.PostLikeKey + strconv.FormatUint(postID, 10)
	count, err := redis.GetInt64(ctx, key)
	if err == nil {
		return count, nil
	}
	realCount, err := s.actionRepo.GetLikeCountByPostID(ctx, postID)
	if err != nil {
		return 0, err
	}
	_ = redis.SetWithExpiration(ctx, key, realCount, cacheExpiration)
	return realCount, nil
}

func (s *postActionServiceImpl) GetPostLikeCounts(ctx context.Context, postIDs []uint64) ([]int64, error) {
	counts := make([]int64, len(postIDs))
	for i, id := range postIDs {
		counts[i], _ = s.GetPostLikeCount(ctx, id)
	}
	return counts, nil
}

func (s *postActionServiceImpl) IsLiked(ctx context.Context, userID, postID uint64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	return s.actionRepo.CheckLikeExists(ctx, userID, postID)
}

func (s *postActionServiceImpl) CollectPost(ctx context.Context, userID, postID uint64) error {
	return s.performAction(s.getPostCheck(ctx, postID), func() error {
		return s.actionRepo.CreateCollection(ctx, &model.Collection{UserID: userID, PostID: postID, CreatedAt: time.Now()})
	})
}

func (s *postActionServiceImpl) CancelCollectPost(ctx context.Context, userID, postID uint64) error {
	return s.revokeAction(s.getPostCheck(ctx, postID), func() error {
		return s.actionRepo.DeleteCollection(ctx, userID, postID)
	})
}

func (s *postActionServiceImpl) GetPostCollectionCount(ctx context.Context, postID uint64) (int64, error) {
	key := consts.PostCollectionKey + strconv.FormatUint(postID, 10)
	count, err := redis.GetInt64(ctx, key)
	if err == nil {
		return count, nil
	}
	realCount, err := s.actionRepo.GetCollectionCountByPostID(ctx, postID)
	if err != nil {
		return 0, err
	}
	_ = redis.SetWithExpiration(ctx, key, realCount, cacheExpiration)
	return realCount, nil
}

func (s *postActionServiceImpl) IsCollected(ctx context.Context, userID, postID uint64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	return s.actionRepo.CheckCollectionExists(ctx, userID, postID)
}

func (s *postActionServiceImpl) CreateComment(ctx context.Context, userID uint64, req *dto.CommentCreateDTO) error {
	post, err := s.postRepo.GetPost(ctx, req.PostID)
	if err != nil || post == nil {
		return ErrPostNotFound
	}

	var hdelKeys []string

	for _, mediaDTO := range req.MediaInfo {
		if err := processMedia(ctx, mediaDTO, &hdelKeys); err != nil {
			return err
		}
	}

	var finalRootID uint64
	var finalParentID uint64
	var replyUserID uint64

	if req.ParentID > 0 {
		parent, err := s.actionRepo.GetCommentByID(ctx, req.ParentID)
		if err != nil || parent == nil || parent.Status != CommentStatusApproved {
			return ErrPostCommentNotFound
		}
		if parent.PostID != req.PostID {
			return ErrPostCommentNotFound
		}

		finalParentID = parent.ID
		replyUserID = parent.UserID

		if parent.RootID == 0 {
			finalRootID = parent.ID
		} else {
			finalRootID = parent.RootID
		}
	} else {
		finalRootID = 0
		finalParentID = 0
		replyUserID = 0
	}

	comment := &model.PostComment{
		PostID:        req.PostID,
		Content:       req.Content,
		MediaInfo:     make(model.MediaList, len(req.MediaInfo)),
		UserID:        userID,
		RootID:        finalRootID,
		ParentID:      finalParentID,
		ReplyToUserID: replyUserID,
		Status:        CommentStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := copier.Copy(&comment.MediaInfo, req.MediaInfo); err != nil {
		return err
	}

	if err := s.actionRepo.CreateComment(ctx, comment); err != nil {
		return err
	}

	if len(hdelKeys) > 0 {
		go func() {
			_ = redis.HDel(context.Background(), consts.MediaTempKey, hdelKeys...)
		}()
	}

	return nil
}

func (s *postActionServiceImpl) DeleteComment(ctx context.Context, userID, commentID uint64) error {
	comment, err := s.actionRepo.GetCommentByID(ctx, commentID)
	if err != nil || comment == nil {
		return ErrPostCommentNotFound
	}

	if comment.UserID != userID {
		return UnauthorizedError
	}

	if err = s.actionRepo.DeleteComment(ctx, commentID); err != nil {
		return err
	}

	if len(comment.MediaInfo) > 0 {
		go func() {
			bgCtx := context.Background()
			for _, m := range comment.MediaInfo {
				_ = minio.DeleteFile(bgCtx, m.MediaURL)
			}
			log.Info("comment media resources cleaned up", "commentID", commentID)
		}()
	}

	return nil
}

func (s *postActionServiceImpl) GetPostCommentCount(ctx context.Context, postID uint64) (int64, error) {
	key := consts.PostCommentKey + strconv.FormatUint(postID, 10)
	count, err := redis.GetInt64(ctx, key)
	if err == nil {
		return count, nil
	}
	realCount, err := s.actionRepo.GetCommentCountByPostID(ctx, postID)
	if err != nil {
		return 0, err
	}
	_ = redis.SetWithExpiration(ctx, key, realCount, cacheExpiration)
	return realCount, nil
}

func (s *postActionServiceImpl) GetCommentsByPostID(ctx context.Context, postID uint64, page, pageSize int) ([]*dto.CommentDTO, error) {
	rootComments, err := s.actionRepo.GetRootCommentsByPostID(ctx, postID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}

	var allIDs []uint64
	rootIDs := make([]uint64, 0, len(rootComments))
	for _, rc := range rootComments {
		allIDs = append(allIDs, rc.ID)
		rootIDs = append(rootIDs, rc.ID)
		for _, sc := range rc.SubComments {
			allIDs = append(allIDs, sc.ID)
		}
	}

	currentUserID, _ := ctx.Value("user_id").(uint64)

	likesMap := s.batchGetLikes(ctx, allIDs, rootComments)
	countMap, _ := s.actionRepo.GetSubCommentCounts(ctx, rootIDs)
	isLikedMap := s.batchGetIsLiked(ctx, currentUserID, allIDs)

	res := make([]*dto.CommentDTO, 0, len(rootComments))
	for _, rc := range rootComments {
		rootDTO := s.convertToCommentDTO(rc, likesMap[rc.ID], isLikedMap[rc.ID])
		rootDTO.SubCommentCount = int64(countMap[rc.ID])

		if len(rc.SubComments) > 0 {
			rootDTO.SubComments = make([]*dto.CommentDTO, 0, len(rc.SubComments))
			for _, sc := range rc.SubComments {
				rootDTO.SubComments = append(rootDTO.SubComments, s.convertToCommentDTO(sc, likesMap[sc.ID], isLikedMap[sc.ID]))
			}
		}
		res = append(res, rootDTO)
	}
	return res, nil
}

func (s *postActionServiceImpl) GetSubComments(ctx context.Context, rootID uint64, page, pageSize int) ([]*dto.CommentDTO, error) {
	subs, err := s.actionRepo.GetSubCommentsByRootID(ctx, rootID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}

	subIDs := make([]uint64, 0, len(subs))
	for _, sc := range subs {
		subIDs = append(subIDs, sc.ID)
	}

	currentUserID, _ := ctx.Value("user_id").(uint64)

	likesMap := s.batchGetLikes(ctx, subIDs, subs)
	isLikedMap := s.batchGetIsLiked(ctx, currentUserID, subIDs)

	res := make([]*dto.CommentDTO, 0, len(subs))
	for _, sc := range subs {
		res = append(res, s.convertToCommentDTO(sc, likesMap[sc.ID], isLikedMap[sc.ID]))
	}
	return res, nil
}

func (s *postActionServiceImpl) GetLikedPosts(ctx context.Context, userID uint64, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	ids, err := s.actionRepo.GetLikedPostIDs(ctx, userID, pageSize+1, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return s.expandPostList(ctx, ids, pageSize)
}

func (s *postActionServiceImpl) GetCollectedPosts(ctx context.Context, userID uint64, page, pageSize int) (*dto.PostWaterfallDTO, error) {
	ids, err := s.actionRepo.GetCollectedPostIDs(ctx, userID, pageSize+1, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return s.expandPostList(ctx, ids, pageSize)
}

func (s *postActionServiceImpl) LikeComment(ctx context.Context, userID, commentID uint64) error {
	return s.performAction(s.getCommentCheck(ctx, commentID), func() error {
		return s.actionRepo.CreateCommentLike(ctx, &model.CommentLike{UserID: userID, CommentID: commentID, CreatedAt: time.Now()})
	})
}

func (s *postActionServiceImpl) CancelLikeComment(ctx context.Context, userID, commentID uint64) error {
	return s.revokeAction(s.getCommentCheck(ctx, commentID), func() error {
		return s.actionRepo.DeleteCommentLike(ctx, userID, commentID)
	})
}

func (s *postActionServiceImpl) GetCommentLikeCount(ctx context.Context, commentID uint64) (int64, error) {
	key := consts.PostCommentLikeKey + strconv.FormatUint(commentID, 10)
	count, err := redis.GetInt64(ctx, key)
	if err == nil {
		return count, nil
	}
	realCount, err := s.actionRepo.GetCommentLikeCount(ctx, commentID)
	if err != nil {
		return 0, err
	}
	_ = redis.SetWithExpiration(ctx, key, realCount, cacheExpiration)
	return realCount, nil
}

func (s *postActionServiceImpl) SyncCommentLikesCount(ctx context.Context, commentID uint64, count int) error {
	return s.actionRepo.UpdateCommentLikesCount(ctx, commentID, count)
}

func (s *postActionServiceImpl) TrackPostView(ctx context.Context, userID, postID uint64) error {
	return s.performAction(s.getPostCheck(ctx, postID), func() error {
		return s.actionRepo.CreateView(ctx, &model.PostView{
			PostID:   postID,
			UserID:   userID,
			ViewedAt: time.Now(),
		})
	})
}

func (s *postActionServiceImpl) GetPostViewCount(ctx context.Context, postID uint64) (int64, error) {
	key := consts.PostViewKey + strconv.FormatUint(postID, 10)
	count, err := redis.GetInt64(ctx, key)
	if err == nil {
		return count, nil
	}
	realCount, err := s.actionRepo.GetViewCountByPostID(ctx, postID)
	if err != nil {
		return 0, err
	}
	_ = redis.SetWithExpiration(ctx, key, realCount, cacheExpiration)
	return realCount, nil
}

func (s *postActionServiceImpl) ReportPost(ctx context.Context, userID, postID uint64) error {
	post, err := s.postRepo.GetPost(ctx, postID)
	if err != nil || post == nil {
		return ErrPostNotFound
	}

	reportLockKey := consts.ReportLock + strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(postID, 10)
	set, err := redis.TryLock(ctx, reportLockKey, "1", 24*time.Hour, 0)
	if err != nil || !set {
		return ErrActionDuplicate
	}

	countKey := consts.PostReportKey + strconv.FormatUint(postID, 10)
	_ = redis.Incr(ctx, countKey)
	count, _ := redis.GetInt64(ctx, countKey)
	if count >= 50 {
		_ = s.postRepo.UpdatePostStatus(ctx, postID, 3)
	}
	return nil
}

func (s *postActionServiceImpl) expandPostList(ctx context.Context, ids []uint64, pageSize int) (*dto.PostWaterfallDTO, error) {
	hasMore := len(ids) > pageSize
	if hasMore {
		ids = ids[:pageSize]
	}
	posts, err := s.postRepo.GetPostByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	list := make([]*dto.PostDTO, 0, len(posts))
	for _, post := range posts {
		item := &dto.PostDTO{}
		_ = copier.Copy(item, post)
		_ = copier.Copy(&item.Medias, &post.MediaList)
		if post.User.ID > 0 {
			item.UserID = post.User.ID
			item.Nickname = post.User.UserDetail.Nickname
			item.AvatarURL = minio.GetPublicURL(post.User.UserDetail.AvatarURL)
		}
		for _, m := range item.Medias {
			m.MediaURL = minio.GetPublicURL(m.MediaURL)
			if m.CoverURL != nil {
				url := minio.GetPublicURL(*m.CoverURL)
				m.CoverURL = &url
			}
		}

		item.CreatedAt = post.CreatedAt.Format("2006-01-02 15:04:05")
		item.UpdatedAt = post.UpdatedAt.Format("2006-01-02 15:04:05")
		list = append(list, item)
	}
	return &dto.PostWaterfallDTO{List: list, HasMore: hasMore}, nil
}

func isDuplicateError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	return false
}

func (s *postActionServiceImpl) performAction(checkFunc func() error, repoFunc func() error) error {
	if err := checkFunc(); err != nil {
		return err
	}
	if err := repoFunc(); err != nil {
		if isDuplicateError(err) {
			return ErrActionDuplicate
		}
		return err
	}
	return nil
}

func (s *postActionServiceImpl) revokeAction(checkFunc func() error, repoFunc func() error) error {
	if err := checkFunc(); err != nil {
		return err
	}
	return repoFunc()
}

func (s *postActionServiceImpl) getPostCheck(ctx context.Context, postID uint64) func() error {
	return func() error {
		posts, err := s.postRepo.GetPostByIds(ctx, []uint64{postID})
		if err != nil || len(posts) == 0 {
			return ErrPostNotFound
		}
		return nil
	}
}

func (s *postActionServiceImpl) getCommentCheck(ctx context.Context, commentID uint64) func() error {
	return func() error {
		comment, err := s.actionRepo.GetCommentByID(ctx, commentID)
		if err != nil || comment == nil {
			return ErrPostCommentNotFound
		}
		return nil
	}
}

func (s *postActionServiceImpl) convertToCommentDTO(comment *model.PostComment, likesCount int, isLiked bool) *dto.CommentDTO {
	dtoItem := &dto.CommentDTO{}
	_ = copier.Copy(dtoItem, comment)
	_ = copier.Copy(&dtoItem.MediaInfo, &comment.MediaInfo)

	dtoItem.LikesCount = likesCount
	dtoItem.IsLiked = isLiked

	if comment.User.UserID != 0 {
		dtoItem.Nickname = comment.User.Nickname
		dtoItem.AvatarURL = minio.GetPublicURL(comment.User.AvatarURL)
	}

	if comment.ReplyToUserID != 0 && comment.ReplyUser.UserID != 0 {
		dtoItem.ReplyToNickname = comment.ReplyUser.Nickname
	}

	dtoItem.CreatedAt = comment.CreatedAt.Format("2006-01-02 15:04:05")

	dtoItem.LikesCount = likesCount

	for _, m := range dtoItem.MediaInfo {
		m.MediaURL = minio.GetPublicURL(m.MediaURL)
		if m.CoverURL != nil {
			url := minio.GetPublicURL(*m.CoverURL)
			m.CoverURL = &url
		}
	}
	return dtoItem
}

func (s *postActionServiceImpl) batchGetLikes(ctx context.Context, ids []uint64, comments []*model.PostComment) map[uint64]int {
	likesMap := make(map[uint64]int)
	if len(ids) == 0 {
		return likesMap
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = consts.PostCommentLikeKey + strconv.FormatUint(id, 10)
	}
	cacheData, _ := redis.MGetValue(ctx, keys...)

	for _, comment := range comments {
		key := consts.PostCommentLikeKey + strconv.FormatUint(comment.ID, 10)

		if valStr, ok := cacheData[key]; ok {
			count, _ := strconv.Atoi(valStr)
			likesMap[comment.ID] = count
		} else {
			likesMap[comment.ID] = comment.LikesCount

			go func(c *model.PostComment) {
				k := consts.PostCommentLikeKey + strconv.FormatUint(c.ID, 10)
				_ = redis.SetWithExpiration(context.Background(), k, c.LikesCount, cacheExpiration)
			}(comment)
		}
	}
	return likesMap
}

func (s *postActionServiceImpl) batchGetIsLiked(ctx context.Context, userID uint64, commentIDs []uint64) map[uint64]bool {
	isLikedMap := make(map[uint64]bool)
	if userID == 0 || len(commentIDs) == 0 {
		return isLikedMap
	}

	pipe := redis.GetRdbClient().Pipeline()
	cmds := make(map[uint64]*redisv9.BoolCmd)
	for _, id := range commentIDs {
		key := consts.PostCommentLikeUserSetKey + strconv.FormatUint(id, 10)
		cmds[id] = pipe.SIsMember(ctx, key, userID)
	}
	_, _ = pipe.Exec(ctx)

	var maybeUnlikedIDs []uint64
	for id, cmd := range cmds {
		isLiked, err := cmd.Result()
		if err == nil && isLiked {
			isLikedMap[id] = true
		} else {
			maybeUnlikedIDs = append(maybeUnlikedIDs, id)
		}
	}

	if len(maybeUnlikedIDs) > 0 {
		dbLikedIDs, err := s.actionRepo.GetLikedCommentIDs(ctx, userID, maybeUnlikedIDs)
		if err == nil && len(dbLikedIDs) > 0 {
			likedSet := make(map[uint64]struct{})
			for _, id := range dbLikedIDs {
				likedSet[id] = struct{}{}
				isLikedMap[id] = true
			}

			go func(uid uint64, ids []uint64) {
				bgCtx := context.Background()
				for _, id := range ids {
					key := consts.PostCommentLikeUserSetKey + strconv.FormatUint(id, 10)
					_ = redis.SAdd(bgCtx, key, uid)
					_ = redis.Expire(bgCtx, key, 24*time.Hour)
				}
			}(userID, dbLikedIDs)
		}
	}

	return isLikedMap
}
