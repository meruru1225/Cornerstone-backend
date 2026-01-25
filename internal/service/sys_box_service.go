package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/mongo"
	"Cornerstone/internal/repository"
	"context"
	"errors"
	"time"

	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongoDB "go.mongodb.org/mongo-driver/mongo"
)

type SysBoxService interface {
	GetNotificationList(ctx context.Context, userID uint64, page, pageSize int) ([]*dto.SysBoxDTO, error)
	GetUnreadCount(ctx context.Context, userID uint64) (*dto.SysBoxUnreadDTO, error)
	MarkRead(ctx context.Context, userID uint64, msgID string) error
	MarkAllRead(ctx context.Context, userID uint64) error
}

type sysBoxServiceImpl struct {
	sysBoxRepo mongo.SysBoxRepo
	userRepo   repository.UserRepo
}

func NewSysBoxService(sysBox mongo.SysBoxRepo, user repository.UserRepo) SysBoxService {
	return &sysBoxServiceImpl{
		sysBoxRepo: sysBox,
		userRepo:   user,
	}
}

// GetNotificationList 获取通知列表并补全用户信息
func (s *sysBoxServiceImpl) GetNotificationList(ctx context.Context, userID uint64, page, pageSize int) ([]*dto.SysBoxDTO, error) {
	limit := int64(pageSize)
	offset := int64((page - 1) * pageSize)

	// 从 MongoDB 拉取原始数据
	list, err := s.sysBoxRepo.GetNotificationList(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	res := make([]*dto.SysBoxDTO, 0, len(list))
	for _, m := range list {
		d := &dto.SysBoxDTO{}
		_ = copier.Copy(d, m)
		d.ID = m.ID.Hex()
		d.CreatedAt = m.CreatedAt.UTC().Format(time.RFC3339)

		// 补全发送者信息 (SenderID 为 0 代表系统发送)
		if m.SenderID > 0 {
			user, err := s.userRepo.GetUserHomeInfoById(ctx, m.SenderID)
			if err == nil && user != nil {
				d.SenderName = user.Nickname
				d.AvatarURL = minio.GetPublicURL(user.AvatarURL)
			}
		} else {
			d.SenderName = "系统通知"
		}

		res = append(res, d)
	}

	return res, nil
}

// GetUnreadCount 获取未读数
func (s *sysBoxServiceImpl) GetUnreadCount(ctx context.Context, userID uint64) (*dto.SysBoxUnreadDTO, error) {
	count, err := s.sysBoxRepo.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.SysBoxUnreadDTO{UnreadCount: count}, nil
}

// MarkRead 标记单条已读
func (s *sysBoxServiceImpl) MarkRead(ctx context.Context, userID uint64, msgID string) error {
	objectID, err := primitive.ObjectIDFromHex(msgID)
	if err != nil {
		return ErrParamInvalid
	}

	notice, err := s.sysBoxRepo.GetByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, mongoDB.ErrNoDocuments) {
			return ErrSysBoxNotFound
		}
		return err
	}

	if notice.ReceiverID != userID {
		return UnauthorizedError
	}

	if notice.IsRead {
		return nil
	}

	return s.sysBoxRepo.MarkAsRead(ctx, userID, msgID)
}

// MarkAllRead 一键已读
func (s *sysBoxServiceImpl) MarkAllRead(ctx context.Context, userID uint64) error {
	return s.sysBoxRepo.MarkAllAsRead(ctx, userID)
}
