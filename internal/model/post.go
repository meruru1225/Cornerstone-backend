package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"
)

type Post struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	UserID        uint64    `gorm:"not null;index:idx_user_id" json:"user_id"`
	Title         string    `gorm:"type:varchar(255)" json:"title"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	MediaList     MediaList `gorm:"type:json" json:"media_list"` // 聚合后的媒体字段
	LikesCount    int       `gorm:"not null;default:0" json:"likes_count"`
	CommentsCount int       `gorm:"not null;default:0" json:"comments_count"`
	CollectsCount int       `gorm:"not null;default:0" json:"collects_count"`
	ViewsCount    int       `gorm:"not null;default:0" json:"views_count"`
	Status        int8      `gorm:"not null;default:0" json:"status"` // 0:审核, 1:发布, 2:拒绝
	IsDeleted     bool      `gorm:"type:tinyint(1);not null;default:0" json:"is_deleted"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (Post) TableName() string {
	return "posts"
}

// MediaItem 媒体信息
type MediaItem struct {
	MimeType string  `json:"mime_type"`
	MediaURL string  `json:"url"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Duration float64 `json:"duration"`
	CoverURL *string `json:"cover_url,omitempty"`
}

// MediaList 媒体列表
type MediaList []MediaItem

func (m MediaList) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *MediaList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, m)
}
