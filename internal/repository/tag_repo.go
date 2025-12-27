package repository

import (
	"Cornerstone/internal/model"
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepo interface {
	GetOrCreateTag(ctx context.Context, tagName string, description string) (*model.Tag, error)
	GetOrCreateTags(ctx context.Context, tagNames []string) ([]*model.Tag, error)
}

type tagRepoImpl struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepo {
	return &tagRepoImpl{
		db: db,
	}
}

func (s *tagRepoImpl) GetOrCreateTag(ctx context.Context, tagName string, description string) (*model.Tag, error) {
	tag := model.Tag{
		Name:        tagName,
		Description: &description,
		CreatedAt:   time.Now(),
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&tag).Error
	if err != nil {
		return nil, err
	}
	// 如果记录已存在，查询获取完整数据
	var existingTag model.Tag
	err = s.db.WithContext(ctx).Where("name = ?", tagName).First(&existingTag).Error
	if err != nil {
		return nil, err
	}
	return &existingTag, nil
}

func (s *tagRepoImpl) GetOrCreateTags(ctx context.Context, tagNames []string) ([]*model.Tag, error) {
	// 创建所有标签，使用 OnConflict DoNothing 避免重复创建
	for _, tagName := range tagNames {
		tag := model.Tag{
			Name:      tagName,
			CreatedAt: time.Now(),
		}
		err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&tag).Error
		if err != nil {
			return nil, err
		}
	}

	// 查询所有请求的标签
	var tags []*model.Tag
	err := s.db.WithContext(ctx).Where("name IN ?", tagNames).Find(&tags).Error
	if err != nil {
		return nil, err
	}

	return tags, nil
}
