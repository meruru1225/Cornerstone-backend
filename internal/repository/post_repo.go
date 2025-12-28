package repository

import (
	"Cornerstone/internal/model"
	"context"
	log "log/slog"

	"gorm.io/gorm"
)

type PostRepo interface {
	CreatePost(ctx context.Context, post *model.Post, media []*model.PostMedia) error
	GetPost(ctx context.Context, id uint64) (*model.Post, error)
	GetPostByIds(ctx context.Context, ids []uint64) ([]*model.Post, error)
	GetPostByUserId(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error)
	GetPostSelf(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error)
	GetPostMedias(ctx context.Context, postId uint64) ([]*model.PostMedia, error)
	GetPostTagNames(ctx context.Context, postId uint64) ([]string, error)
	UpdatePost(ctx context.Context, post *model.Post, media []*model.PostMedia) error
	UpdatePostStatus(ctx context.Context, id uint64, status int) error
	DeletePost(ctx context.Context, id uint64) error
	UpsertPostTag(ctx context.Context, postID uint64, tagName string) error
}

type PostRepoImpl struct {
	db *gorm.DB
}

func NewPostRepo(db *gorm.DB) PostRepo {
	return &PostRepoImpl{
		db: db,
	}
}

func (s *PostRepoImpl) CreatePost(ctx context.Context, post *model.Post, media []*model.PostMedia) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		return s.createPostAssociations(tx, post.ID, media)
	})
}

func (s *PostRepoImpl) GetPost(ctx context.Context, id uint64) (*model.Post, error) {
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

func (s *PostRepoImpl) GetPostByUserId(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error) {
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

func (s *PostRepoImpl) GetPostSelf(ctx context.Context, userId uint64, limit, offset int) ([]*model.Post, error) {
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

func (s *PostRepoImpl) GetPostMedias(ctx context.Context, postId uint64) ([]*model.PostMedia, error) {
	var medias []*model.PostMedia

	err := s.db.WithContext(ctx).
		Select("id", "url", "media_type").
		Where("post_id = ?", postId).
		Order("sort_order ASC").
		Find(&medias).Error

	if err != nil {
		return nil, err
	}
	return medias, nil
}

func (s *PostRepoImpl) GetPostTagNames(ctx context.Context, postId uint64) ([]string, error) {
	var tagNames []string
	err := s.db.WithContext(ctx).
		Table("post_tags").
		Select("tags.name").
		Joins("JOIN tags ON post_tags.tag_id = tags.id").
		Where("post_tags.post_id = ?", postId).
		Find(&tagNames).Error
	if err != nil {
		return nil, err
	}

	return tagNames, nil
}

func (s *PostRepoImpl) UpdatePost(ctx context.Context, post *model.Post, media []*model.PostMedia) error {
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
		return s.createPostAssociations(tx, post.ID, media)
	})
}

func (s *PostRepoImpl) UpdatePostStatus(ctx context.Context, id uint64, status int) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Update("status", status).Error
}

func (s *PostRepoImpl) DeletePost(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).Update("is_deleted", true).Error
}

func (s *PostRepoImpl) UpsertPostTag(ctx context.Context, postID uint64, tagName string) error {
	var newTag model.Tag
	err := s.db.WithContext(ctx).Select("id").Where("name = ?", tagName).First(&newTag).Error
	if err != nil {
		log.Warn("tag not found", "name", tagName)
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.Where("post_id = ?", postID).Delete(&model.PostTag{}).Error; err != nil {
			return err
		}
		postTag := model.PostTag{
			PostID: postID,
			TagID:  newTag.ID,
		}
		return tx.Create(&postTag).Error
	})
}

func (s *PostRepoImpl) createPostAssociations(tx *gorm.DB, postID uint64, media []*model.PostMedia) error {
	if len(media) > 0 {
		for i := range media {
			media[i].PostID = postID
			media[i].ID = 0
		}
		if err := tx.Create(media).Error; err != nil {
			return err
		}
	}
	return nil
}
