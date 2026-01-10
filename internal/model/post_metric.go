package model

import (
	"time"
)

type PostMetric struct {
	ID            uint64    `gorm:"primaryKey"`
	PostID        uint64    `gorm:"not null;index:idx_post_date,unique"`
	MetricDate    time.Time `gorm:"not null;index:idx_post_date,unique;column:metric_date"`
	TotalLikes    int       `gorm:"not null;default:0"`
	TotalComments int       `gorm:"not null;default:0"`
	TotalCollects int       `gorm:"not null;default:0"`
	TotalViews    int       `gorm:"not null;default:0"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (PostMetric) TableName() string {
	return "post_daily_metrics"
}
