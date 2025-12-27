package model

import (
	"time"
)

type PostDailyMetric struct {
	ID          uint64    `gorm:"primaryKey"`
	PostID      uint64    `gorm:"not null;index:idx_post_date,unique" json:"postId"`
	MetricDate  time.Time `gorm:"not null;index:idx_post_date,unique;column:metric_date" json:"metricDate"`
	NewLikes    int       `gorm:"not null;default:0" json:"newLikes"`
	NewComments int       `gorm:"not null;default:0" json:"newComments"`
	NewCollects int       `gorm:"not null;default:0" json:"newCollects"`
	NewViews    int       `gorm:"not null;default:0" json:"newViews"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (PostDailyMetric) TableName() string {
	return "post_daily_metrics"
}
