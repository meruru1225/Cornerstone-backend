package repository

import (
	"Cornerstone/internal/model"
	"context"

	"gorm.io/gorm"
)

type PostRepo interface {
	CreatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, Tags []*model.PostTag) error
	GetPost(ctx context.Context, id uint64) (*model.Post, error)
	GetPostByIds(ctx context.Context, ids []uint64) ([]*model.Post, error)
	UpdatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, Tags []*model.PostTag) error
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

func (s PostRepoImpl) CreatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, Tags []*model.PostTag) error {
	if len(media) == 0 {
		return s.db.WithContext(ctx).Create(post).Error
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		if err := tx.Create(media).Error; err != nil {
			return err
		}
		if err := tx.Create(Tags).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s PostRepoImpl) GetPost(ctx context.Context, id uint64) (*model.Post, error) {
	var post model.Post
	err := s.db.WithContext(ctx).Preload("User").Preload("Media").Preload("Comments").Preload("Tags").First(&post, id).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s PostRepoImpl) GetPostByIds(ctx context.Context, ids []uint64) ([]*model.Post, error) {
	var posts []*model.Post
	err := s.db.WithContext(ctx).Preload("User").Preload("Media").Preload("Comments").Preload("Tags").Where("id IN ?", ids).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (s PostRepoImpl) UpdatePost(ctx context.Context, post *model.Post, media []*model.PostMedia, Tags []*model.PostTag) error {
	err := s.db.WithContext(ctx).Delete(&model.PostMedia{}, "post_id = ?", post.ID).Error
	if err != nil {
		return err
	}
	err = s.db.WithContext(ctx).Delete(&model.Tag{}, "post_id = ?", post.ID).Error
	if err != nil {
		return err
	}
	if len(media) == 0 {
		return s.db.WithContext(ctx).Updates(post).Error
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Updates(post).Error; err != nil {
			return err
		}
		if err = tx.Create(media).Error; err != nil {
			return err
		}
		if err = tx.Create(Tags).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s PostRepoImpl) DeletePost(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Update("is_deleted", true).Error
}
