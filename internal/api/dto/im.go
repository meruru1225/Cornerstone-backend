package dto

import "time"

// SendMessageReq 发送消息请求体
type SendMessageReq struct {
	ConversationID uint64          `json:"conversation_id"`
	TargetUserID   uint64          `json:"target_user_id"`
	MsgType        int             `json:"msg_type" binding:"required"` // 1-正常消息 2-音频消息 3-撤回消息
	Content        string          `json:"content" binding:"required"`
	Payload        []MediasBaseDTO `json:"payload"`
}

// MessageDTO 消息明细响应
type MessageDTO struct {
	ID             string          `json:"id,omitempty"`
	ConversationID uint64          `json:"conversation_id"`
	SenderID       uint64          `json:"sender_id"`
	MsgType        int             `json:"msg_type"`
	Content        string          `json:"content"`
	Payload        []MediasBaseDTO `json:"payload"`
	Seq            uint64          `json:"seq"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// ConversationDTO 会话列表项响应
type ConversationDTO struct {
	ConversationID uint64    `json:"conversation_id"`
	Type           int8      `json:"type"`    // 1-单聊, 2-群聊
	PeerID         uint64    `json:"peer_id"` // 对手方ID (单聊有效)
	LastMsgContent string    `json:"last_msg_content"`
	LastMsgType    int8      `json:"last_msg_type"`
	LastSenderID   uint64    `json:"last_sender_id"`
	LastMessageAt  time.Time `json:"last_message_at"`
	UnreadCount    uint64    `json:"unread_count"`
	IsMuted        bool      `json:"is_muted"`
	IsPinned       bool      `json:"is_pinned"`

	CoverURL string `json:"cover_url"`
	Title    string `json:"title"`

	// 进度字段
	MyReadSeq   uint64 `json:"my_read_seq"`
	PeerReadSeq uint64 `json:"peer_read_seq"`
	MaxSeq      uint64 `json:"max_seq"`
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
