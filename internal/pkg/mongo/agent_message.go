package mongo

import (
	"time"
)

type AgentMessage struct {
	ID             string    `bson:"_id,omitempty" json:"id"`
	ConversationID string    `bson:"conversation_id" json:"conversationId"`
	SenderID       uint64    `bson:"sender_id" json:"senderId"` // 0 - Agent, 1 - Guest, Other - Custom User
	Content        string    `bson:"content" json:"content"`
	Seq            uint64    `bson:"seq" json:"seq"`
	CreatedAt      time.Time `bson:"created_at" json:"createdAt"`
}

type TokenUsage struct {
	PromptTokens     int `bson:"prompt_tokens"`
	CompletionTokens int `bson:"completion_tokens"`
	TotalTokens      int `bson:"total_tokens"`
}
