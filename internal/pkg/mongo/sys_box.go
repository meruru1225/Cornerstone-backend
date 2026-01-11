package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SysBoxModel 系统通知模型
type SysBoxModel struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ReceiverID uint64             `bson:"receiver_id" json:"receiverId"` // 消息接收者ID
	SenderID   uint64             `bson:"sender_id" json:"senderId"`     // 动作发起者ID (系统通知可为0)
	Type       int8               `bson:"type" json:"type"`              // 通知类型: 1-帖子点赞, 2-帖子收藏, 3-帖子评论, 4-评论点赞, 5-被关注
	TargetID   uint64             `bson:"target_id" json:"targetId"`     // 关联的目标ID (如帖子ID、评论ID)
	Content    string             `bson:"content" json:"content"`        // 通知文案预览或评论片段
	Payload    map[string]any     `bson:"payload" json:"payload"`        // 额外元数据 (可选，如帖子标题快照)
	IsRead     bool               `bson:"is_read" json:"isRead"`         // 是否已读
	CreatedAt  time.Time          `bson:"created_at" json:"createdAt"`   // 创建时间
}
