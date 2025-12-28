package model

import (
	"time"
)

type Post struct {
	ID             uint64    `gorm:"primaryKey"`
	UserID         uint64    `gorm:"not null;index:idx_user_id" json:"user_id"`
	Title          string    `gorm:"type:varchar(255)" json:"title"`
	Content        string    `gorm:"not null" json:"content"`
	LikesCount     int       `gorm:"not null;default:0" json:"likes_count"`
	CommentsCount  int       `gorm:"not null;default:0" json:"comments_count"`
	CollectsCount  int       `gorm:"not null;default:0" json:"collects_count"`
	Status         int8      `gorm:"not null;default:0" json:"status"` // 0:审核中, 1:已发布, 2:拒绝, 3:待人工
	IsDeleted      bool      `gorm:"type:tinyint(1);not null;default:0" json:"is_deleted"`
	ContentVersion int       `gorm:"not null;default:1" json:"content_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 关联关系
	User  User        `gorm:"foreignKey:UserID;references:ID"`
	Media []PostMedia `gorm:"foreignKey:PostID;references:ID"`
}

func (Post) TableName() string {
	return "posts"
}
