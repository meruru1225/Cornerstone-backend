package model

import "time"

// Conversation 会话主表
type Conversation struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Type           int8      `gorm:"not null;default:1" json:"type"`              // 1-单聊, 2-群聊
	PeerKey        string    `gorm:"uniqueIndex;type:varchar(64)" json:"peerKey"` // uid1_uid2
	MaxMsgSeq      uint64    `gorm:"not null;default:0" json:"maxMsgSeq"`         // 序列号
	LastMsgContent string    `gorm:"type:varchar(255)" json:"lastMsgContent"`
	LastMsgType    int8      `gorm:"not null;default:1" json:"lastMsgType"`
	LastSenderID   uint64    `gorm:"not null;default:0" json:"lastSenderId"`
	LastMessageAt  time.Time `gorm:"index" json:"lastMessageAt"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (Conversation) TableName() string { return "conversations" }

// ConversationMember 会话成员表
type ConversationMember struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID uint64    `gorm:"uniqueIndex:idx_conv_user" json:"conversationId"`
	UserID         uint64    `gorm:"uniqueIndex:idx_conv_user;index" json:"userId"`
	ReadMsgSeq     uint64    `gorm:"not null;default:0" json:"readMsgSeq"` // 已读进度
	IsMuted        int8      `gorm:"not null;default:0" json:"isMuted"`
	IsPinned       int8      `gorm:"not null;default:0" json:"isPinned"`
	IsVisible      int8      `gorm:"not null;default:1;index" json:"isVisible"` // 会话列表可见性
	JoinedAt       time.Time `json:"joinedAt"`

	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID" json:"conversation"`

	// 虚拟字段：仅读不写，存储 SQL 计算结果
	UnreadCount uint64 `gorm:"->" json:"unreadCount"`
}

func (ConversationMember) TableName() string { return "conversation_members" }
