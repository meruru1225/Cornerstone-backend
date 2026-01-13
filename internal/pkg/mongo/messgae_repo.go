package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageRepo interface {
	SaveMessage(ctx context.Context, msg *Message) error
	GetHistory(ctx context.Context, convID uint64, lastSeq uint64, pageSize int) ([]*Message, error)
	GetMessageBySeq(ctx context.Context, convID uint64, seq uint64) (*Message, error)
}

type messageRepoImpl struct {
	col *mongo.Collection
}

func NewMessageRepo(db *mongo.Database) MessageRepo {
	return &messageRepoImpl{
		col: db.Collection("message"),
	}
}

// SaveMessage 将消息存入 MongoDB
func (s *messageRepoImpl) SaveMessage(ctx context.Context, msg *Message) error {
	_, err := s.col.InsertOne(ctx, msg)
	return err
}

// GetHistory 历史消息查询逻辑
// lastSeq 为当前页面最旧一条消息的序号。如果是第一页，传 0。
func (s *messageRepoImpl) GetHistory(ctx context.Context, convID uint64, lastSeq uint64, pageSize int) ([]*Message, error) {
	// 基础过滤：指定会话 ID
	filter := bson.M{"conversation_id": convID}

	// 游标过滤：如果是拉取历史记录，找比当前最旧序号 (lastSeq) 更小的消息
	if lastSeq > 0 {
		filter["seq"] = bson.M{"$lt": lastSeq}
	}

	// 排序与限制，按照 seq 降序排列 (最新的在前)，限制返回条数
	findOptions := options.Find().
		SetSort(bson.D{{Key: "seq", Value: -1}}).
		SetLimit(int64(pageSize))

	cursor, err := s.col.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	// 解析结果
	var messages []*Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetMessageBySeq 精确查询
func (s *messageRepoImpl) GetMessageBySeq(ctx context.Context, convID uint64, seq uint64) (*Message, error) {
	var msg Message
	filter := bson.M{
		"conversation_id": convID,
		"seq":             seq,
	}
	err := s.col.FindOne(ctx, filter).Decode(&msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
