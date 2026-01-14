package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostRepo interface {
	CreatePost(ctx context.Context, post *model.Post) error
	GetPost(ctx context.Context, id uint64) (*model.Post, error)
	GetPostByIds(ctx context.Context, ids []uint64) ([]*model.Post, error)
	GetPostByUserId(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error)
	GetPostSelf(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error)
	GetPostsByStatusCursor(ctx context.Context, status int, lastID uint64, limit int) ([]*model.Post, error)
	DeletePost(ctx context.Context, id uint64) error
	UpdatePostContent(ctx context.Context, post *model.Post) error
	UpdatePostStatus(ctx context.Context, id uint64, status int) error
	UpdatePostCounts(ctx context.Context, pid uint64, likes int64, comments int64, collects int64, views int64) error
	GetPostMedias(ctx context.Context, postId uint64) (model.MediaList, error)
	GetPostTagNames(ctx context.Context, postId uint64) ([]string, error)
	SyncPostMainTag(ctx context.Context, postID uint64, tagName string) error
}

type PostRepoImpl struct {
	db *gorm.DB
}

func NewPostRepo(db *gorm.DB) PostRepo {
	return &PostRepoImpl{
		db: db,
	}
}

// CreatePost 创建笔记
func (s *PostRepoImpl) CreatePost(ctx context.Context, post *model.Post) error {
	return s.db.WithContext(ctx).Create(post).Error
}

// GetPost 获取单个笔记
func (s *PostRepoImpl) GetPost(ctx context.Context, id uint64) (*model.Post, error) {
	var post model.Post
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id")
		}).
		Preload("User.UserDetail", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id", "nickname", "avatar_url")
		}).
		Where("id = ? AND is_deleted = ? AND status = ?", id, false, 1).
		First(&post).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

// GetPostByIds 批量获取笔记
func (s *PostRepoImpl) GetPostByIds(ctx context.Context, ids []uint64) ([]*model.Post, error) {
	var posts []*model.Post
	if len(ids) == 0 {
		return posts, nil
	}
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id")
		}).
		Preload("User.UserDetail", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id", "nickname", "avatar_url")
		}).
		Where("id IN ? AND is_deleted = ? AND status = ?", ids, false, 1).
		Find(&posts).Error

	return posts, err
}

// GetPostByUserId 获取他人主页已发布的笔记
func (s *PostRepoImpl) GetPostByUserId(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error) {
	var posts []*model.Post
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id")
		}).
		Preload("User.UserDetail", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id", "nickname", "avatar_url")
		}).
		Where("user_id = ? AND is_deleted = ? AND status = ?", userId, false, 1).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&posts).Error

	return posts, err
}

// GetPostSelf 获取自己所有的笔记
func (s *PostRepoImpl) GetPostSelf(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error) {
	var posts []*model.Post
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id")
		}).
		Preload("User.UserDetail", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id", "nickname", "avatar_url")
		}).
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&posts).Error

	return posts, err
}

// GetPostsByStatusCursor 根据笔记状态分页获取笔记
func (s *PostRepoImpl) GetPostsByStatusCursor(ctx context.Context, status int, lastID uint64, limit int) ([]*model.Post, error) {
	db := s.db.WithContext(ctx).Where("status = ?", status)

	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}

	var posts []*model.Post
	err := db.Order("id DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

// GetPostMedias 从笔记 JSON 字段中直接提取媒体列表
func (s *PostRepoImpl) GetPostMedias(ctx context.Context, postId uint64) (model.MediaList, error) {
	var post model.Post
	err := s.db.WithContext(ctx).
		Select("media_list").
		Where("id = ?", postId).
		First(&post).Error

	return post.MediaList, err
}

// GetPostTagNames 获取笔记关联的标签名称
func (s *PostRepoImpl) GetPostTagNames(ctx context.Context, postId uint64) ([]string, error) {
	var tagNames []string
	err := s.db.WithContext(ctx).
		Table("post_tags").
		Select("tags.name").
		Joins("JOIN tags ON post_tags.tag_id = tags.id").
		Where("post_tags.post_id = ?", postId).
		Find(&tagNames).Error

	return tagNames, err
}

// UpdatePostContent 更新内容与媒体
func (s *PostRepoImpl) UpdatePostContent(ctx context.Context, post *model.Post) error {
	updateData := map[string]interface{}{
		"title":      post.Title,
		"content":    post.Content,
		"media_list": post.MediaList,
		"status":     0,
	}
	// 仅限作者本人修改，且不涉及任何关联表操作，性能极高
	return s.db.WithContext(ctx).Model(&model.Post{}).
		Where("id = ? AND user_id = ?", post.ID, post.UserID).
		Updates(updateData).Error
}

// UpdatePostStatus 更新笔记状态
func (s *PostRepoImpl) UpdatePostStatus(ctx context.Context, id uint64, status int) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Update("status", status).Error
}

func (s *PostRepoImpl) UpdatePostCounts(ctx context.Context, pid uint64, likes int64, comments int64, collects int64, views int64) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", pid).Updates(map[string]interface{}{
		"likes_count":    likes,
		"comments_count": comments,
		"collects_count": collects,
		"views_count":    views,
	}).Error
}

// DeletePost 逻辑删除
func (s *PostRepoImpl) DeletePost(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Update("is_deleted", true).Error
}

// SyncPostMainTag 同步 AI 计算出的主标签
func (s *PostRepoImpl) SyncPostMainTag(ctx context.Context, postID uint64, tagName string) error {
	var tag model.Tag
	err := s.db.WithContext(ctx).Select("id").Where("name = ?", tagName).First(&tag).Error
	if err != nil {
		return nil
	}

	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "post_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"tag_id"}),
	}).Create(&model.PostTag{
		PostID: postID,
		TagID:  tag.ID,
	}).Error
}
