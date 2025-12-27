package model

import (
	"time"
)

type PostView struct {
	ID       uint64    `gorm:"primaryKey"`
	PostID   uint64    `gorm:"not null;index:idx_post_id" json:"postId"`
	UserID   uint64    `gorm:"not null" json:"userId"`
	ViewedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"viewedAt"`
}

func (PostView) TableName() string {
	return "post_views"
}
