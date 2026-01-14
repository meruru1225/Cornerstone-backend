package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type UserRepo interface {
	GetUserById(ctx context.Context, id uint64) (*model.User, error)
	GetUserByIds(ctx context.Context, ids []uint64) ([]*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*model.User, error)
	GetUserHomeInfoById(ctx context.Context, id uint64) (*model.UserDetail, error)
	GetUserSimpleInfoByIds(ctx context.Context, ids []uint64) ([]*model.UserDetail, error)
	CreateUser(ctx context.Context, user *model.User, detail *model.UserDetail, roles *[]*model.UserRole) error
	UpdateUser(ctx context.Context, user *model.User) error
	UpdateUserIsBan(ctx context.Context, id uint64, isBan bool) (int64, error)
	UpdateUserDetail(ctx context.Context, detail *model.UserDetail) error
	UpdateUserFollowCount(ctx context.Context, id uint64, followerCount int64, followingCount int64) error
	DeleteUser(ctx context.Context, id uint64) error
}

type UserRepoImpl struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &UserRepoImpl{db: db}
}

func (s *UserRepoImpl) GetUserById(ctx context.Context, id uint64) (*model.User, error) {
	user := &model.User{}
	result := s.db.WithContext(ctx).
		Preload("UserDetail").
		Preload("UserRoles").
		First(user, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return user, nil
}

func (s *UserRepoImpl) GetUserByIds(ctx context.Context, ids []uint64) ([]*model.User, error) {
	users := make([]*model.User, 0)
	result := s.db.WithContext(ctx).
		Preload("UserDetail").
		Preload("UserRoles").
		Where("id IN ?", ids).
		Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	return users, nil
}

func (s *UserRepoImpl) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	result := s.db.WithContext(ctx).
		Preload("UserDetail").
		Preload("UserRoles").
		Where("username = ?", username).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return user, nil
}

func (s *UserRepoImpl) GetUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	user := &model.User{}
	result := s.db.WithContext(ctx).
		Preload("UserDetail").
		Preload("UserRoles").
		Where("phone = ?", phone).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return user, nil
}

func (s *UserRepoImpl) GetUserHomeInfoById(ctx context.Context, id uint64) (*model.UserDetail, error) {
	user := &model.UserDetail{}
	result := s.db.WithContext(ctx).
		Select("user_id", "nickname", "gender", "region", "avatar_url", "bio").
		Where("user_id = ?", id).
		First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return user, nil
}

func (s *UserRepoImpl) GetUserSimpleInfoByIds(ctx context.Context, ids []uint64) ([]*model.UserDetail, error) {
	users := make([]*model.UserDetail, 0)
	result := s.db.WithContext(ctx).
		Select("user_id", "nickname", "avatar_url", "bio").
		Where("user_id IN ?", ids).
		Find(&users)

	if result.Error != nil {
		return nil, result.Error
	}

	return users, nil
}

func (s *UserRepoImpl) CreateUser(ctx context.Context, user *model.User, detail *model.UserDetail, roles *[]*model.UserRole) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if result := tx.Create(user); result.Error != nil {
			return result.Error
		}

		detail.UserID = user.ID
		if result := tx.Create(detail); result.Error != nil {
			return result.Error
		}

		for _, role := range *roles {
			role.UserID = user.ID
		}
		if result := tx.Create(roles); result.Error != nil {
			return result.Error
		}

		return nil
	})
}

func (s *UserRepoImpl) UpdateUser(ctx context.Context, user *model.User) error {
	result := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *UserRepoImpl) UpdateUserIsBan(ctx context.Context, id uint64, isBan bool) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("is_ban", isBan)

	return result.RowsAffected, result.Error
}

func (s *UserRepoImpl) UpdateUserDetail(ctx context.Context, detail *model.UserDetail) error {
	result := s.db.WithContext(ctx).Model(&model.UserDetail{}).Where("user_id = ?", detail.UserID).Updates(detail)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *UserRepoImpl) UpdateUserFollowCount(ctx context.Context, id uint64, followerCount int64, followingCount int64) error {
	result := s.db.WithContext(ctx).Model(&model.UserDetail{}).Where("user_id = ?", id).Updates(map[string]interface{}{
		"followers_count": followerCount,
		"following_count": followingCount,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *UserRepoImpl) DeleteUser(ctx context.Context, id uint64) error {
	usernamePlaceholder := fmt.Sprintf("deleted_%d_%d", id, time.Now().Unix())

	userUpdate := model.User{
		IsDelete: true,
		Username: &usernamePlaceholder,
		Password: nil,
		Phone:    nil,
	}

	detailUpdate := model.UserDetail{
		Nickname:  "已注销用户",
		Bio:       nil,
		AvatarURL: "default_avatar.png",
		Region:    nil,
		Birthday:  nil,
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userFields := []string{"is_delete", "username", "password", "phone"}
		if result := tx.Model(&model.User{}).Where("id = ?", id).Select(userFields).Updates(userUpdate); result.Error != nil {
			return result.Error
		}

		detailFields := []string{"nickname", "bio", "avatar_url", "region", "birthday"}
		if result := tx.Model(&model.UserDetail{}).Where("user_id = ?", id).Select(detailFields).Updates(detailUpdate); result.Error != nil {
			return result.Error
		}

		result := s.db.WithContext(ctx).Model(&model.UserRole{}).Where("user_id = ?", id).Delete(&model.UserRole{})
		if result.Error != nil {
			return result.Error
		}

		result = s.db.WithContext(ctx).Model(&model.UserFollow{}).Where("follower_id = ?", id).Delete(&model.UserFollow{})
		if result.Error != nil {
			return result.Error
		}

		result = s.db.WithContext(ctx).Model(&model.UserFollow{}).Where("following_id = ?", id).Delete(&model.UserFollow{})
		if result.Error != nil {
			return result.Error
		}

		return nil
	})
}
