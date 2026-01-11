package mongo

import "time"

// Message MongoDB 消息明细模型
type Message struct {
	ID             string    `bson:"_id,omitempty" json:"id"`               // MongoDB 自动生成的 ObjectID
	ConversationID uint64    `bson:"conversation_id" json:"conversationId"` // 关联 MySQL 的会话 ID
	SenderID       uint64    `bson:"sender_id" json:"senderId"`             // 发送者 UID
	MsgType        int       `bson:"msg_type" json:"msgType"`               // 1-文本, 2-图片, 3-语音, 4-视频, 5-文件, 6-撤回
	Content        string    `bson:"content" json:"content"`                // 文本内容或消息预览
	Payload        MMap      `bson:"payload,omitempty" json:"payload"`      // 结构化附件（如 URL, 宽高, 时长等）
	Seq            uint64    `bson:"seq" json:"seq"`                        // 该消息在会话中的唯一绝对序号 (来自 MySQL)
	ReplyTo        uint64    `bson:"reply_to,omitempty" json:"replyTo"`     // 被回复的消息 Seq
	CreatedAt      time.Time `bson:"created_at" json:"createdAt"`           // 消息发送时间
}

// MMap 定义一个通用的 Map 类型，方便处理动态的 Payload
type MMap map[string]interface{}
