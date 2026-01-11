package dto

import "time"

// SendMessageReq 发送消息请求体
type SendMessageReq struct {
	ConversationID uint64                 `json:"conversationId" binding:"required"`
	MsgType        int                    `json:"msgType" binding:"required"` // 1-文本, 2-图片...
	Content        string                 `json:"content" binding:"required"`
	Payload        map[string]interface{} `json:"payload"`
}

// MessageDTO 消息明细响应
type MessageDTO struct {
	ID             string                 `json:"id"`
	ConversationID uint64                 `json:"conversationId"`
	SenderID       uint64                 `json:"senderId"`
	MsgType        int                    `json:"msgType"`
	Content        string                 `json:"content"`
	Payload        map[string]interface{} `json:"payload"`
	Seq            uint64                 `json:"seq"`
	CreatedAt      time.Time              `json:"createdAt"`
}

// ConversationDTO 会话列表项响应
type ConversationDTO struct {
	ConversationID uint64    `json:"conversationId"`
	Type           int8      `json:"type"`   // 1-单聊, 2-群聊
	PeerID         uint64    `json:"peerId"` // 对手方ID (单聊有效)
	LastMsgContent string    `json:"lastMsgContent"`
	LastMsgType    int8      `json:"lastMsgType"`
	LastSenderID   uint64    `json:"lastSenderId"`
	LastMessageAt  time.Time `json:"lastMessageAt"`
	UnreadCount    uint64    `json:"unreadCount"`
	IsMuted        bool      `json:"isMuted"`
	IsPinned       bool      `json:"isPinned"`
}

// ReadReceiptDTO 已读回执推送
type ReadReceiptDTO struct {
	ConversationID uint64 `json:"conversationId"`
	UserID         uint64 `json:"userId"`
	ReadSeq        uint64 `json:"readSeq"`
	Type           string `json:"type"`
}

// MarkAsReadReq 标记为已读请求
type MarkAsReadReq struct {
	ConversationID uint64 `json:"conversationId" binding:"required"`
	Sequence       uint64 `json:"sequence" binding:"required"` // 客户端当前看到的最后一条消息序号
}
