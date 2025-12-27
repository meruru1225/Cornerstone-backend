package model

import "time"

type UserMetrics struct {
	ID             uint64    `gorm:"primaryKey"`
	UserID         uint64    `gorm:"not null"`
	MetricDate     time.Time `gorm:"type:date;not null;uniqueIndex:idx_user_date,columns:user_id,metric_date"`
	TotalFollowers int       `gorm:"type:int;not null;default:0"`
	CreatedAt      time.Time
}

func (UserMetrics) TableName() string {
	return "user_metrics"
}
