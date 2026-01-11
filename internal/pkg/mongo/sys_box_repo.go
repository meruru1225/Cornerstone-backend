package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SysBoxRepo interface {
	CreateNotification(ctx context.Context, msg *SysBoxModel) error
	GetNotificationList(ctx context.Context, userID uint64, limit, offset int64) ([]*SysBoxModel, error)
	MarkAsRead(ctx context.Context, userID uint64, msgID string) error
	MarkAllAsRead(ctx context.Context, userID uint64) error
	GetUnreadCount(ctx context.Context, userID uint64) (int64, error)
}

type sysBoxRepoImpl struct {
	col *mongo.Collection
}

func NewSysBoxRepo(db *mongo.Database) SysBoxRepo {
	return &sysBoxRepoImpl{
		col: db.Collection("sys_box"),
	}
}

// CreateNotification 插入新通知
func (r *sysBoxRepoImpl) CreateNotification(ctx context.Context, msg *SysBoxModel) error {
	_, err := r.col.InsertOne(ctx, msg)
	return err
}

// GetNotificationList 分页获取用户的通知列表 (按时间倒序)
func (r *sysBoxRepoImpl) GetNotificationList(ctx context.Context, userID uint64, limit, offset int64) ([]*SysBoxModel, error) {
	filter := bson.M{"receiver_id": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var list []*SysBoxModel
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// MarkAsRead 标记单条通知为已读
func (r *sysBoxRepoImpl) MarkAsRead(ctx context.Context, userID uint64, msgID string) error {
	objectID, err := primitive.ObjectIDFromHex(msgID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objectID, "receiver_id": userID}
	update := bson.M{"$set": bson.M{"is_read": true}}
	_, err = r.col.UpdateOne(ctx, filter, update)
	return err
}

// MarkAllAsRead 一键清除未读 (将用户所有未读通知标记为已读)
func (r *sysBoxRepoImpl) MarkAllAsRead(ctx context.Context, userID uint64) error {
	filter := bson.M{"receiver_id": userID, "is_read": false}
	update := bson.M{"$set": bson.M{"is_read": true}}
	_, err := r.col.UpdateMany(ctx, filter, update)
	return err
}

// GetUnreadCount 获取用户的未读通知总数
func (r *sysBoxRepoImpl) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	filter := bson.M{"receiver_id": userID, "is_read": false}
	return r.col.CountDocuments(ctx, filter)
}
