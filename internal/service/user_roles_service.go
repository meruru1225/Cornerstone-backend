package service

import (
	"Cornerstone/internal/repository"
	"context"
)

type UserRolesService interface {
	AddRoleToUser(ctx context.Context, userId uint64, roleId uint64) error
	DeleteRoleFromUser(ctx context.Context, userId uint64, roleId uint64) error
}

type UserRolesServiceImpl struct {
	userRolesRepo repository.UserRolesRepo
}

func NewUserRolesService(userRolesRepo repository.UserRolesRepo) UserRolesService {
	return &UserRolesServiceImpl{userRolesRepo: userRolesRepo}
}

func (s *UserRolesServiceImpl) AddRoleToUser(ctx context.Context, userId uint64, roleId uint64) error {
	hasRole, err := s.userRolesRepo.GetUserHasRole(ctx, userId, roleId)
	if err != nil {
		return err
	}
	if hasRole {
		return ErrUserHasRole
	}
	return s.userRolesRepo.AddRoleToUser(ctx, userId, roleId)
}

func (s *UserRolesServiceImpl) DeleteRoleFromUser(ctx context.Context, userId uint64, roleId uint64) error {
	return s.userRolesRepo.DeleteRoleFromUser(ctx, userId, roleId)
}
