package model

import (
	"time"
)

type PostMedia struct {
	ID        uint64    `gorm:"primaryKey"`
	PostID    uint64    `gorm:"not null;index:idx_post_id_sort" json:"postId"`
	FileType  string    `gorm:"type:varchar(64);not null" json:"fileType"` // e.g., image/jpeg, video/mp4
	MediaURL  string    `gorm:"type:varchar(512);not null" json:"mediaUrl"`
	SortOrder int8      `gorm:"not null;default:0" json:"sortOrder"`
	Width     int       `gorm:"not null;default:0" json:"width"`
	Height    int       `gorm:"not null;default:0" json:"height"`
	CoverURL  *string   `gorm:"type:varchar(512)" json:"coverUrl"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (PostMedia) TableName() string {
	return "post_media"
}
