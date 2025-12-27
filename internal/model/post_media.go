package model

import (
	"time"
)

type PostMedia struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	PostID    uint64    `gorm:"not null;index:idx_post_id_sort" json:"post_id"`
	FileType  string    `gorm:"type:varchar(64);not null" json:"file_type"` // e.g., image/jpeg, video/mp4
	MediaURL  string    `gorm:"type:varchar(512);not null" json:"media_url"`
	SortOrder int8      `gorm:"not null;default:0" json:"sort_order"`
	Width     int       `gorm:"not null;default:0" json:"width"`
	Height    int       `gorm:"not null;default:0" json:"height"`
	Duration  int       `gorm:"not null;default:0" json:"duration"`
	CoverURL  *string   `gorm:"type:varchar(512)" json:"cover_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PostMedia) TableName() string {
	return "post_media"
}
