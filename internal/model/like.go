package model

import (
	"time"
)

type Like struct {
	UserID    uint64    `gorm:"primaryKey" json:"userId"`
	PostID    uint64    `gorm:"primaryKey;index:idx_post_id" json:"postId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (Like) TableName() string {
	return "likes"
}
