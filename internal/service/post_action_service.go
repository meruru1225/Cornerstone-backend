package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/copier"
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
	IsCommentLiked(ctx context.Context, userID, commentID uint64) (bool, error)
	GetCommentLikeCount(ctx context.Context, commentID uint64) (int64, error)
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
	return s.performAction(ctx, postID, consts.PostLikeKey, s.getPostCheck(ctx, postID), func() error {
		return s.actionRepo.CreateLike(ctx, &model.Like{UserID: userID, PostID: postID, CreatedAt: time.Now()})
	})
}

func (s *postActionServiceImpl) CancelLikePost(ctx context.Context, userID, postID uint64) error {
	return s.revokeAction(ctx, postID, consts.PostLikeKey, s.getPostCheck(ctx, postID), func() (int64, error) {
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
	return s.performAction(ctx, postID, consts.PostCollectionKey, s.getPostCheck(ctx, postID), func() error {
		return s.actionRepo.CreateCollection(ctx, &model.Collection{UserID: userID, PostID: postID, CreatedAt: time.Now()})
	})
}

func (s *postActionServiceImpl) CancelCollectPost(ctx context.Context, userID, postID uint64) error {
	return s.revokeAction(ctx, postID, consts.PostCollectionKey, s.getPostCheck(ctx, postID), func() (int64, error) {
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
	check := func() error {
		if err := s.getPostCheck(ctx, req.PostID)(); err != nil {
			return err
		}

		if req.RootID > 0 {
			rootComment, err := s.actionRepo.GetCommentByID(ctx, req.RootID)
			if err != nil || rootComment == nil {
				return ErrPostCommentNotFound
			}
			if rootComment.PostID != req.PostID {
				return ErrPostCommentNotFound
			}
			if rootComment.RootID != 0 {
				return ErrPostCommentNotFound
			}
		}

		if req.ParentID > 0 {
			if err := s.getCommentCheck(ctx, req.ParentID)(); err != nil {
				return ErrPostCommentNotFound
			}
		}
		return nil
	}

	return s.performAction(ctx, req.PostID, consts.PostCommentKey, check, func() error {
		comment := &model.PostComment{}
		_ = copier.Copy(comment, req)
		comment.UserID = userID
		comment.CreatedAt = time.Now()
		comment.UpdatedAt = time.Now()
		return s.actionRepo.CreateComment(ctx, comment)
	})
}

// DeleteComment 删除评论
func (s *postActionServiceImpl) DeleteComment(ctx context.Context, userID, commentID uint64) error {
	var targetPostID uint64

	check := func() error {
		comment, err := s.actionRepo.GetCommentByID(ctx, commentID)
		if err != nil || comment == nil {
			return ErrPostCommentNotFound
		}
		if comment.UserID != userID {
			return UnauthorizedError
		}
		targetPostID = comment.PostID
		return nil
	}

	if err := check(); err != nil {
		return err
	}

	return s.revokeAction(ctx, targetPostID, consts.PostCommentKey, func() error { return nil }, func() (int64, error) {
		return s.actionRepo.DeleteComment(ctx, commentID)
	})
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

	var res []*dto.CommentDTO
	for _, rc := range rootComments {
		rootDTO := s.convertToCommentDTO(ctx, rc)

		subCount, _ := s.actionRepo.GetSubCommentCountByRootID(ctx, rc.ID)
		rootDTO.SubCommentCount = subCount

		if subCount > 0 {
			subs, _ := s.actionRepo.GetSubCommentsByRootID(ctx, rc.ID, 3, 0)
			for _, sc := range subs {
				subDTO := s.convertToCommentDTO(ctx, sc)
				rootDTO.SubComments = append(rootDTO.SubComments, subDTO)
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

	var res []*dto.CommentDTO
	for _, sc := range subs {
		res = append(res, s.convertToCommentDTO(ctx, sc))
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
	return s.performAction(ctx, commentID, consts.PostCommentLikeKey, s.getCommentCheck(ctx, commentID), func() error {
		return s.actionRepo.CreateCommentLike(ctx, &model.CommentLike{UserID: userID, CommentID: commentID, CreatedAt: time.Now()})
	})
}

func (s *postActionServiceImpl) CancelLikeComment(ctx context.Context, userID, commentID uint64) error {
	return s.revokeAction(ctx, commentID, consts.PostCommentLikeKey, s.getCommentCheck(ctx, commentID), func() (int64, error) {
		return s.actionRepo.DeleteCommentLike(ctx, userID, commentID)
	})
}

func (s *postActionServiceImpl) IsCommentLiked(ctx context.Context, userID, commentID uint64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	return s.actionRepo.CheckCommentLikeExists(ctx, userID, commentID)
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

func (s *postActionServiceImpl) expandPostList(ctx context.Context, ids []uint64, pageSize int) (*dto.PostWaterfallDTO, error) {
	hasMore := len(ids) > pageSize
	if hasMore {
		ids = ids[:pageSize]
	}

	posts, err := s.postRepo.GetPostByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	var list []*dto.PostDTO
	for _, post := range posts {
		item := &dto.PostDTO{}
		_ = copier.Copy(item, post)
		_ = copier.Copy(&item.Medias, &post.MediaList)

		if post.User.ID > 0 {
			item.UserID = post.User.ID
			item.Nickname = post.User.UserDetail.Nickname
			item.AvatarURL = post.User.UserDetail.AvatarURL
		}

		item.CreatedAt = post.CreatedAt.Format("2006-01-02 15:04:05")
		item.UpdatedAt = post.UpdatedAt.Format("2006-01-02 15:04:05")

		list = append(list, item)
	}

	return &dto.PostWaterfallDTO{
		List:    list,
		HasMore: hasMore,
	}, nil
}

func isDuplicateError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	return false
}

// performAction 辅助函数，用于执行点赞、收藏等操作
func (s *postActionServiceImpl) performAction(
	ctx context.Context,
	targetID uint64,
	redisKeyPrefix string,
	checkFunc func() error,
	repoFunc func() error,
) error {
	if err := checkFunc(); err != nil {
		return err
	}

	if err := repoFunc(); err != nil {
		if isDuplicateError(err) {
			return ErrActionDuplicate
		}
		return err
	}

	_ = redis.Incr(ctx, redisKeyPrefix+strconv.FormatUint(targetID, 10))
	_ = redis.SAdd(ctx, consts.PostDirtyKey, targetID)
	return nil
}

// revokeAction 辅助函数，用于取消点赞、收藏等操作
func (s *postActionServiceImpl) revokeAction(
	ctx context.Context,
	targetID uint64,
	redisKeyPrefix string,
	checkFunc func() error,
	repoFunc func() (int64, error),
) error {
	if err := checkFunc(); err != nil {
		return err
	}

	affected, err := repoFunc()
	if err != nil {
		return err
	}

	if affected > 0 {
		_ = redis.Decr(ctx, redisKeyPrefix+strconv.FormatUint(targetID, 10))
		_ = redis.SAdd(ctx, consts.PostDirtyKey, targetID)
	} else {
		return ErrActionDuplicate
	}
	return nil
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

func (s *postActionServiceImpl) convertToCommentDTO(ctx context.Context, c *model.PostComment) *dto.CommentDTO {
	dtoItem := &dto.CommentDTO{}
	_ = copier.Copy(dtoItem, c)
	_ = copier.Copy(&dtoItem.MediaInfo, &c.MediaInfo)

	user, _ := s.userRepo.GetUserHomeInfoById(ctx, c.UserID)
	if user != nil {
		dtoItem.Nickname = user.Nickname
		dtoItem.AvatarURL = user.AvatarURL
	}

	if c.ReplyToUserID > 0 {
		target, _ := s.userRepo.GetUserById(ctx, c.ReplyToUserID)
		if target != nil {
			dtoItem.ReplyToNickname = target.UserDetail.Nickname
		}
	}

	dtoItem.CreatedAt = c.CreatedAt.Format("2006-01-02 15:04:05")

	commentLikeKey := consts.PostCommentLikeKey + strconv.FormatUint(c.ID, 10)
	if val, err := redis.GetInt64(ctx, commentLikeKey); err == nil {
		dtoItem.LikesCount = int(val)
	}
	return dtoItem
}
