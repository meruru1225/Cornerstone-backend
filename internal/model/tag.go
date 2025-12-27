package model

import "time"

type Tag struct {
	ID          uint64  `gorm:"primaryKey"`
	Name        string  `gorm:"type:varchar(50);not null;uniqueIndex:idx_tag_name"`
	Description *string `gorm:"type:varchar(255)"` // 默认可为空
	CreatedAt   time.Time
}

func (Tag) TableName() string {
	return "tags"
}
