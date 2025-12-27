package model

import (
	"time"
)

type Collection struct {
	UserID    uint64    `gorm:"primaryKey" json:"userId"`
	PostID    uint64    `gorm:"primaryKey;index:idx_post_id" json:"postId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (Collection) TableName() string {
	return "collections"
}
