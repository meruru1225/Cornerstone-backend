package model

import (
	"time"
)

type PostComment struct {
	ID            uint64             `gorm:"primaryKey"`
	PostID        uint64             `gorm:"not null;index:idx_post_id" json:"postId"`
	UserID        uint64             `gorm:"not null" json:"userId"`
	Content       string             `gorm:"type:varchar(1000);not null" json:"content"`
	MediaInfo     []CommentMediaItem `gorm:"type:json;serializer:json" json:"mediaInfo"`
	RootID        uint64             `gorm:"not null;default:0;index:idx_root_id" json:"rootId"` // 0表示这是一级评论
	ParentID      uint64             `gorm:"not null;default:0" json:"parentId"`                 // 0表示这是直接评论帖子
	ReplyToUserID uint64             `gorm:"not null;default:0" json:"replyToUserId"`            // 0表示无回复目标
	LikesCount    int                `gorm:"not null;default:0" json:"likesCount"`
	IsDeleted     bool               `gorm:"type:tinyint(1);not null;default:0" json:"isDeleted"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

func (PostComment) TableName() string {
	return "post_comments"
}

type CommentMediaItem struct {
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
	MimeType string `json:"mimeType"`
}
