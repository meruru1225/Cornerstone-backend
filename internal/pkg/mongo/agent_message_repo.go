package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AgentMessageRepo interface {
	SaveMessage(ctx context.Context, msg *AgentMessage) error
	GetHistory(ctx context.Context, convID string, limit int) ([]*AgentMessage, error)
}

type agentMessageRepoImpl struct {
	col *mongo.Collection
}

func NewAgentMessageRepo(db *mongo.Database) AgentMessageRepo {
	return &agentMessageRepoImpl{
		col: db.Collection("agent_messages"),
	}
}

// SaveMessage 直接存储
func (s *agentMessageRepoImpl) SaveMessage(ctx context.Context, msg *AgentMessage) error {
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	_, err := s.col.InsertOne(ctx, msg)
	return err
}

// GetHistory 纯按时间线拉取最近 20 条
func (s *agentMessageRepoImpl) GetHistory(ctx context.Context, convID string, limit int) ([]*AgentMessage, error) {
	if limit <= 0 {
		limit = 20
	}

	filter := bson.M{"conversation_id": convID}

	findOptions := options.Find().
		SetSort(bson.D{
			{Key: "created_at", Value: -1},
			{Key: "_id", Value: -1},
		}).
		SetLimit(int64(limit))

	cursor, err := s.col.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	messages := make([]*AgentMessage, 0)
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	// 反转消息列表，保证消息从旧到新排列
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
