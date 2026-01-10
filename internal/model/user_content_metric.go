package model

import (
	"time"
)

// UserContentMetric 用户内容表现快照模型
type UserContentMetric struct {
	ID            uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID        uint64    `gorm:"not null;uniqueIndex:idx_user_date;column:user_id" json:"userId"`
	MetricDate    time.Time `gorm:"not null;type:date;uniqueIndex:idx_user_date;column:metric_date" json:"metricDate"`
	TotalLikes    int       `gorm:"not null;default:0;column:total_likes" json:"totalLikes"`
	TotalCollects int       `gorm:"not null;default:0;column:total_collects" json:"totalCollects"`
	TotalComments int       `gorm:"not null;default:0;column:total_comments" json:"totalComments"`
	TotalViews    int       `gorm:"not null;default:0;column:total_views" json:"totalViews"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (UserContentMetric) TableName() string {
	return "user_content_metrics"
}
