package mongo

import (
	"time"
)

// Message MongoDB 消息明细模型
type Message struct {
	ID             string    `bson:"_id,omitempty" json:"id"`               // MongoDB 自动生成的 ObjectID
	ConversationID uint64    `bson:"conversation_id" json:"conversationId"` // 关联 MySQL 的会话 ID
	SenderID       uint64    `bson:"sender_id" json:"senderId"`             // 发送者 UID
	MsgType        int       `bson:"msg_type" json:"msgType"`               // 1-正常消息, 2-音频消息, 3-撤回消息
	Content        string    `bson:"content" json:"content"`                // 文本内容或消息预览
	Payload        []Payload `bson:"payload,omitempty" json:"payload"`      // 结构化附件（如 URL, 宽高, 时长等）
	Seq            uint64    `bson:"seq" json:"seq"`                        // 该消息在会话中的唯一绝对序号 (来自 MySQL)
	ReplyTo        uint64    `bson:"reply_to,omitempty" json:"replyTo"`     // 被回复的消息 Seq
	CreatedAt      time.Time `bson:"created_at" json:"createdAt"`           // 消息发送时间
}

// Payload 附件
type Payload struct {
	MimeType string  `bson:"mime_type" json:"mime_type"`
	MediaURL string  `bson:"url" json:"url"`
	Width    int     `bson:"width" json:"width"`
	Height   int     `bson:"height" json:"height"`
	Duration float64 `bson:"duration" json:"duration"`
	CoverURL string  `bson:"cover_url,omitempty" json:"cover_url"`
}
