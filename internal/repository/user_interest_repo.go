package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserInterestRepo interface {
	SaveUserInterests(ctx context.Context, data *model.UserInterestTags) error
	GetUserInterests(ctx context.Context, userID uint64) (*model.UserInterestTags, error)
}

type userInterestRepoImpl struct {
	db *gorm.DB
}

func NewUserInterestRepository(db *gorm.DB) UserInterestRepo {
	return &userInterestRepoImpl{db: db}
}

// SaveUserInterests 保存用户的兴趣画像
func (r *userInterestRepoImpl) SaveUserInterests(ctx context.Context, data *model.UserInterestTags) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"interests", "updated_at"}),
	}).Create(data).Error
}

// GetUserInterests 根据用户 ID 获取其兴趣画像快照
func (r *userInterestRepoImpl) GetUserInterests(ctx context.Context, userID uint64) (*model.UserInterestTags, error) {
	var interests model.UserInterestTags
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&interests).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &interests, nil
}
