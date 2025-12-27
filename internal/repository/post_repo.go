package repository

import (
	"Cornerstone/internal/model"
	"context"

	"gorm.io/gorm"
)

type PostRepo interface {
	CreatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, tags []*model.PostTag) error
	GetPost(ctx context.Context, id uint64) (*model.Post, error)
	GetPostByIds(ctx context.Context, ids []uint64) ([]*model.Post, error)
	GetPostByUserId(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error)
	GetPostSelf(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error)
	UpdatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, tags []*model.PostTag) error
	DeletePost(ctx context.Context, id uint64) error
}

type PostRepoImpl struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepo {
	return &PostRepoImpl{
		db: db,
	}
}

func (s PostRepoImpl) CreatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, tags []*model.PostTag) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		return s.createPostAssociations(tx, post.ID, media, tags)
	})
}

func (s PostRepoImpl) GetPost(ctx context.Context, id uint64) (*model.Post, error) {
	var post model.Post
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id")
		}).
		Preload("User.UserDetail", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id", "nickname", "avatar_url")
		}).
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&post).Error

	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s PostRepoImpl) GetPostByIds(ctx context.Context, ids []uint64) ([]*model.Post, error) {
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
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("id IN ? AND is_deleted = ?", ids, false).
		Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (s PostRepoImpl) GetPostByUserId(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error) {
	var posts []*model.Post
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id")
		}).
		Preload("User.UserDetail", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id", "nickname", "avatar_url")
		}).
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("user_id = ? AND is_deleted = ? AND status = ?", userId, false, 1).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (s PostRepoImpl) GetPostSelf(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error) {
	var posts []*model.Post
	err := s.db.WithContext(ctx).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id")
		}).
		Preload("User.UserDetail", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id", "nickname", "avatar_url")
		}).
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("user_id = ? AND is_deleted = ?", userId, false).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&posts).Error

	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (s PostRepoImpl) UpdatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, tags []*model.PostTag) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", post.ID).Delete(&model.PostMedia{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", post.ID).Delete(&model.PostTag{}).Error; err != nil {
			return err
		}
		if err := tx.Select("title", "content", "status", "updated_at").Updates(post).Error; err != nil {
			return err
		}
		return s.createPostAssociations(tx, post.ID, media, tags)
	})
}

func (s PostRepoImpl) DeletePost(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Update("is_deleted", true).Error
}

func (s PostRepoImpl) createPostAssociations(tx *gorm.DB, postID uint64, media []*model.PostMedia, tags []*model.PostTag) error {
	if len(media) > 0 {
		for i := range media {
			media[i].PostID = postID
			media[i].ID = 0
		}
		if err := tx.Create(media).Error; err != nil {
			return err
		}
	}

	if len(tags) > 0 {
		for i := range tags {
			tags[i].PostID = postID
		}
		if err := tx.Create(tags).Error; err != nil {
			return err
		}
	}
	return nil
}
