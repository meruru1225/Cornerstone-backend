package dto

import "time"

// SendMessageReq 发送消息请求体
type SendMessageReq struct {
	ConversationID uint64                 `json:"conversation_id"`
	TargetUserID   uint64                 `json:"target_user_id"`
	MsgType        int                    `json:"msg_type" binding:"required"` // 1-文本, 2-图片...
	Content        string                 `json:"content" binding:"required"`
	Payload        map[string]interface{} `json:"payload"`
}

// MessageDTO 消息明细响应
type MessageDTO struct {
	ID             string                 `json:"id,omitempty"`
	ConversationID uint64                 `json:"conversation_id"`
	SenderID       uint64                 `json:"sender_id"`
	MsgType        int                    `json:"msg_type"`
	Content        string                 `json:"content"`
	Payload        map[string]interface{} `json:"payload"`
	Seq            uint64                 `json:"seq"`
	CreatedAt      time.Time              `json:"createdAt"`
}

// ConversationDTO 会话列表项响应
type ConversationDTO struct {
	ConversationID uint64    `json:"conversation_id"`
	Type           int8      `json:"type"`    // 1-单聊, 2-群聊
	PeerID         uint64    `json:"peer_id"` // 对手方ID (单聊有效)
	LastMsgContent string    `json:"last_msg_content"`
	LastMsgType    int8      `json:"last_msg_type"`
	LastSenderID   uint64    `json:"last_sender_id"`
	LastMessageAt  time.Time `json:"lastMessageAt"`
	UnreadCount    uint64    `json:"unreadCount"`
	IsMuted        bool      `json:"isMuted"`
	IsPinned       bool      `json:"isPinned"`
}

// ReadReceiptDTO 已读回执推送
type ReadReceiptDTO struct {
	ConversationID uint64 `json:"conversation_id"`
	UserID         uint64 `json:"user_id"`
	ReadSeq        uint64 `json:"read_seq"`
	Type           string `json:"type"`
}

// MarkAsReadReq 标记为已读请求
type MarkAsReadReq struct {
	ConversationID uint64 `json:"conversation_id" binding:"required"`
	Sequence       uint64 `json:"sequence" binding:"required"` // 客户端当前看到的最后一条消息序号
}
