package model

import (
	"time"
)

type CommentLike struct {
	UserID    uint64    `gorm:"primaryKey" json:"userId"`
	CommentID uint64    `gorm:"primaryKey;index:idx_comment_id" json:"commentId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (CommentLike) TableName() string {
	return "comment_likes"
}
