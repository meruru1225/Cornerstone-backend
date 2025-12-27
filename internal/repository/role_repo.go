package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type RoleRepo interface {
	GetRoleByIDs(ctx context.Context, id []uint64) (*[]*model.Role, error)
}

type RoleRepoImpl struct {
	db *gorm.DB
}

func NewRoleRepo(db *gorm.DB) RoleRepo {
	return &RoleRepoImpl{
		db: db,
	}
}

func (s *RoleRepoImpl) GetRoleByIDs(ctx context.Context, id []uint64) (*[]*model.Role, error) {
	roles := make([]*model.Role, 0)
	result := s.db.WithContext(ctx).Model(&model.Role{}).Where("id IN ?", id).Find(&roles)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		} else {
			return nil, result.Error
		}
	}
	return &roles, nil
}
