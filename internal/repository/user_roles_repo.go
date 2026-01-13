package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type UserRolesRepo interface {
	GetRoles(ctx context.Context) ([]*model.Role, error)
	GetUserRoles(ctx context.Context, userId uint64) ([]*model.Role, error)
	GetUserHasRole(ctx context.Context, userId uint64, roleId uint64) (bool, error)
	AddRoleToUser(ctx context.Context, userId uint64, roleId uint64) error
	DeleteRoleFromUser(ctx context.Context, userId uint64, roleId uint64) error
}

type UserRolesRepoImpl struct {
	db *gorm.DB
}

func NewUserRolesRepo(db *gorm.DB) UserRolesRepo {
	return &UserRolesRepoImpl{db: db}
}

func (s *UserRolesRepoImpl) GetRoles(ctx context.Context) ([]*model.Role, error) {
	var roles []*model.Role
	err := s.db.WithContext(ctx).Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *UserRolesRepoImpl) GetUserRoles(ctx context.Context, userId uint64) ([]*model.Role, error) {
	var roles []*model.Role
	err := s.db.WithContext(ctx).
		Table("roles").
		Select("roles.*").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userId).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *UserRolesRepoImpl) GetUserHasRole(ctx context.Context, userId uint64, roleId uint64) (bool, error) {
	var userRole model.UserRole
	result := s.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Where("role_id = ?", roleId).
		First(&userRole)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (s *UserRolesRepoImpl) AddRoleToUser(ctx context.Context, userId uint64, roleId uint64) error {
	return s.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Create(&model.UserRole{
			UserID: userId,
			RoleID: roleId,
		}).Error
}

func (s *UserRolesRepoImpl) DeleteRoleFromUser(ctx context.Context, userId uint64, roleId uint64) error {
	return s.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("user_id = ?", userId).
		Where("role_id = ?", roleId).
		Delete(&model.UserRole{}).Error
}
