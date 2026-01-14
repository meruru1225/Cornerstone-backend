package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserFollowRepo interface {
	GetUserFollowers(ctx context.Context, userID uint64, limit, offset int) ([]*model.UserFollow, error)
	GetUserFollowing(ctx context.Context, userID uint64, limit, offset int) ([]*model.UserFollow, error)
	GetUserFollowerCount(ctx context.Context, userID uint64) (int64, error)
	GetUserFollowingCount(ctx context.Context, userID uint64) (int64, error)
	GetUserFollow(ctx context.Context, userID uint64, followingID uint64) (*model.UserFollow, error)
	CreateUserFollow(ctx context.Context, userFollow *model.UserFollow) error
	DeleteUserFollow(ctx context.Context, userFollow *model.UserFollow) error
}

type UserFollowRepoImpl struct {
	db *gorm.DB
}

func NewUserFollowRepo(db *gorm.DB) UserFollowRepo {
	return &UserFollowRepoImpl{db: db}
}

// GetUserFollowers 获取用户的粉丝列表
func (s *UserFollowRepoImpl) GetUserFollowers(ctx context.Context, userID uint64, limit, offset int) ([]*model.UserFollow, error) {
	var userFollows []*model.UserFollow
	result := s.db.WithContext(ctx).
		Where("following_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&userFollows)

	if result.Error != nil {
		return nil, result.Error
	}
	return userFollows, nil
}

// GetUserFollowing 获取用户的关注列表
func (s *UserFollowRepoImpl) GetUserFollowing(ctx context.Context, userID uint64, limit, offset int) ([]*model.UserFollow, error) {
	var userFollows []*model.UserFollow
	result := s.db.WithContext(ctx).
		Where("follower_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&userFollows)

	if result.Error != nil {
		return nil, result.Error

	}
	return userFollows, nil
}

// GetUserFollowerCount 获取用户的粉丝数量
func (s *UserFollowRepoImpl) GetUserFollowerCount(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	result := s.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("following_id = ?", userID).
		Count(&count)

	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// GetUserFollowingCount 获取用户的关注数量
func (s *UserFollowRepoImpl) GetUserFollowingCount(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	result := s.db.WithContext(ctx).
		Model(&model.UserFollow{}).
		Where("follower_id = ?", userID).
		Count(&count)

	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// GetUserFollow 获取用户的关注关系
func (s *UserFollowRepoImpl) GetUserFollow(ctx context.Context, userID uint64, followingID uint64) (*model.UserFollow, error) {
	var userFollow model.UserFollow
	result := s.db.WithContext(ctx).
		Where("follower_id = ? AND following_id = ?", userID, followingID).
		First(&userFollow)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &userFollow, nil
}

// CreateUserFollow 创建用户的关注关系
func (s *UserFollowRepoImpl) CreateUserFollow(ctx context.Context, userFollow *model.UserFollow) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(userFollow).Error
}

// DeleteUserFollow 删除用户的关注关系
func (s *UserFollowRepoImpl) DeleteUserFollow(ctx context.Context, userFollow *model.UserFollow) error {
	return s.db.WithContext(ctx).Delete(userFollow).Error
}
