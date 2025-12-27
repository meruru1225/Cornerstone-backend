package model

import (
	"time"
)

type Post struct {
	ID            uint64    `gorm:"primaryKey"`
	UserID        uint64    `gorm:"not null;index:idx_user_id" json:"userId"`
	Title         string    `gorm:"type:varchar(255)" json:"title"`
	Content       string    `gorm:"not null" json:"content"`
	LikesCount    int       `gorm:"not null;default:0" json:"likesCount"`
	CommentsCount int       `gorm:"not null;default:0" json:"commentsCount"`
	CollectsCount int       `gorm:"not null;default:0" json:"collectsCount"`
	Status        int8      `gorm:"not null;default:0" json:"status"` // 0:审核中, 1:已发布, 2:拒绝, 3:待人工
	IsDeleted     bool      `gorm:"type:tinyint(1);not null;default:0" json:"isDeleted"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// 关联关系
	User  User        `gorm:"foreignKey:UserID;references:ID"`
	Media []PostMedia `gorm:"foreignKey:PostID;references:ID"`
}

func (Post) TableName() string {
	return "posts"
}
