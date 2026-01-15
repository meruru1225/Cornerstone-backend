package repository

import (
	"Cornerstone/internal/model"
	"context"

	"gorm.io/gorm"
)

type PostActionRepo interface {
	CreateLike(ctx context.Context, like *model.Like) error
	DeleteLike(ctx context.Context, userID, postID uint64) error
	CheckLikeExists(ctx context.Context, userID, postID uint64) (bool, error)
	GetLikedPostIDs(ctx context.Context, userID uint64, limit, offset int) ([]uint64, error)

	CreateCollection(ctx context.Context, collection *model.Collection) error
	DeleteCollection(ctx context.Context, userID, postID uint64) error
	CheckCollectionExists(ctx context.Context, userID, postID uint64) (bool, error)
	GetCollectedPostIDs(ctx context.Context, userID uint64, limit, offset int) ([]uint64, error)

	CreateComment(ctx context.Context, comment *model.PostComment) error
	DeleteComment(ctx context.Context, commentID uint64) error
	UpdateCommentStatus(ctx context.Context, commentID uint64, status int8) error
	UpdateCommentLikesCount(ctx context.Context, commentID uint64, count int) error
	GetCommentByID(ctx context.Context, commentID uint64) (*model.PostComment, error)
	GetRootCommentsByPostID(ctx context.Context, postID uint64, limit, offset int) ([]*model.PostComment, error)
	GetSubCommentsByRootID(ctx context.Context, rootID uint64, limit, offset int) ([]*model.PostComment, error)
	GetSubCommentCountByRootID(ctx context.Context, rootID uint64) (int64, error)

	CreateCommentLike(ctx context.Context, cl *model.CommentLike) error
	DeleteCommentLike(ctx context.Context, userID, commentID uint64) error
	CheckCommentLikeExists(ctx context.Context, userID, commentID uint64) (bool, error)
	GetCommentLikeCount(ctx context.Context, commentID uint64) (int64, error)

	CreateView(ctx context.Context, view *model.PostView) error

	GetLikeCountByPostID(ctx context.Context, postID uint64) (int64, error)
	GetCollectionCountByPostID(ctx context.Context, postID uint64) (int64, error)
	GetCommentCountByPostID(ctx context.Context, postID uint64) (int64, error)
	GetUserTotalLikes(ctx context.Context, userID uint64) (int64, error)
	GetViewCountByPostID(ctx context.Context, postID uint64) (int64, error)
	GetUserTotalCollects(ctx context.Context, userID uint64) (int64, error)
	GetUserTotalComments(ctx context.Context, userID uint64) (int64, error)
	GetUserTotalViews(ctx context.Context, userID uint64) (int64, error)
}

type PostActionRepoImpl struct {
	db *gorm.DB
}

func NewPostActionRepo(db *gorm.DB) PostActionRepo {
	return &PostActionRepoImpl{db}
}

func (s *PostActionRepoImpl) CreateLike(ctx context.Context, like *model.Like) error {
	return s.db.WithContext(ctx).Create(like).Error
}

func (s *PostActionRepoImpl) DeleteLike(ctx context.Context, userID, postID uint64) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.Like{}).Error
}

func (s *PostActionRepoImpl) CheckLikeExists(ctx context.Context, userID, postID uint64) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.Like{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error
	return count > 0, err
}

func (s *PostActionRepoImpl) GetLikedPostIDs(ctx context.Context, userID uint64, limit, offset int) ([]uint64, error) {
	var postIDs []uint64
	err := s.db.WithContext(ctx).Model(&model.Like{}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Pluck("post_id", &postIDs).Error
	return postIDs, err
}

func (s *PostActionRepoImpl) CreateCollection(ctx context.Context, collection *model.Collection) error {
	return s.db.WithContext(ctx).Create(collection).Error
}

func (s *PostActionRepoImpl) DeleteCollection(ctx context.Context, userID, postID uint64) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.Collection{}).Error
}

func (s *PostActionRepoImpl) CheckCollectionExists(ctx context.Context, userID, postID uint64) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.Collection{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error
	return count > 0, err
}

func (s *PostActionRepoImpl) GetCollectedPostIDs(ctx context.Context, userID uint64, limit, offset int) ([]uint64, error) {
	var postIDs []uint64
	err := s.db.WithContext(ctx).Model(&model.Collection{}).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Pluck("post_id", &postIDs).Error
	return postIDs, err
}

func (s *PostActionRepoImpl) CreateComment(ctx context.Context, comment *model.PostComment) error {
	return s.db.WithContext(ctx).Create(comment).Error
}

func (s *PostActionRepoImpl) DeleteComment(ctx context.Context, commentID uint64) error {
	return s.db.WithContext(ctx).Model(&model.PostComment{}).
		Where("(id = ? OR root_id = ?) AND is_deleted = ?", commentID, commentID, false).
		Update("is_deleted", true).Error
}

func (s *PostActionRepoImpl) UpdateCommentStatus(ctx context.Context, commentID uint64, status int8) error {
	return s.db.WithContext(ctx).Model(&model.PostComment{}).
		Where("id = ?", commentID).
		Update("status", status).Error
}

func (s *PostActionRepoImpl) UpdateCommentLikesCount(ctx context.Context, commentID uint64, count int) error {
	return s.db.WithContext(ctx).Model(&model.PostComment{}).
		Where("id = ?", commentID).
		Update("likes_count", count).Error
}

func (s *PostActionRepoImpl) GetCommentByID(ctx context.Context, commentID uint64) (*model.PostComment, error) {
	var comment model.PostComment
	err := s.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", commentID, false).
		First(&comment).Error
	return &comment, err
}

// GetRootCommentsByPostID 分页获取帖子的顶级评论
func (s *PostActionRepoImpl) GetRootCommentsByPostID(ctx context.Context, postID uint64, limit, offset int) ([]*model.PostComment, error) {
	var comments []*model.PostComment
	err := s.db.WithContext(ctx).
		Where("post_id = ? AND root_id = ? AND is_deleted = ?", postID, 0, false).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&comments).Error
	return comments, err
}

// GetSubCommentsByRootID 获取某个根评论下的子评论
func (s *PostActionRepoImpl) GetSubCommentsByRootID(ctx context.Context, rootID uint64, limit, offset int) ([]*model.PostComment, error) {
	var comments []*model.PostComment
	err := s.db.WithContext(ctx).
		Where("root_id = ? AND is_deleted = ?", rootID, false).
		Order("created_at ASC").
		Limit(limit).Offset(offset).
		Find(&comments).Error
	return comments, err
}

// GetSubCommentCountByRootID 获取某个根评论下的回复总数
func (s *PostActionRepoImpl) GetSubCommentCountByRootID(ctx context.Context, rootID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.PostComment{}).
		Where("root_id = ? AND is_deleted = ?", rootID, false).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) CreateCommentLike(ctx context.Context, cl *model.CommentLike) error {
	return s.db.WithContext(ctx).Create(cl).Error
}

func (s *PostActionRepoImpl) DeleteCommentLike(ctx context.Context, userID, commentID uint64) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Delete(&model.CommentLike{}).Error
}

func (s *PostActionRepoImpl) CheckCommentLikeExists(ctx context.Context, userID, commentID uint64) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Count(&count).Error
	return count > 0, err
}

func (s *PostActionRepoImpl) GetCommentLikeCount(ctx context.Context, commentID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("comment_id = ?", commentID).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) CreateView(ctx context.Context, view *model.PostView) error {
	return s.db.WithContext(ctx).Create(view).Error
}

func (s *PostActionRepoImpl) GetLikeCountByPostID(ctx context.Context, postID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.Like{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) GetCollectionCountByPostID(ctx context.Context, postID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.Collection{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) GetCommentCountByPostID(ctx context.Context, postID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.PostComment{}).
		Where("post_id = ? AND is_deleted = ?", postID, false).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) GetUserTotalLikes(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Table("likes").
		Joins("JOIN posts ON likes.post_id = posts.id").
		Where("posts.user_id = ? AND posts.is_deleted = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) GetUserTotalCollects(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Table("collections").
		Joins("JOIN posts ON collections.post_id = posts.id").
		Where("posts.user_id = ? AND posts.is_deleted = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) GetUserTotalComments(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Table("post_comments").
		Joins("JOIN posts ON post_comments.post_id = posts.id").
		Where("posts.user_id = ? AND posts.is_deleted = ?", userID, false).
		Where("post_comments.status = ? AND post_comments.is_deleted = ?", 1, false).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) GetUserTotalViews(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Table("post_views").
		Joins("JOIN posts ON post_views.post_id = posts.id").
		Where("posts.user_id = ? AND posts.is_deleted = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (s *PostActionRepoImpl) GetViewCountByPostID(ctx context.Context, postID uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.PostView{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}
