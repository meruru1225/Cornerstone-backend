package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostMetricRepo interface {
	SaveOrUpdateMetric(ctx context.Context, metric *model.PostMetric) error
	GetPostMetricsBy7Days(ctx context.Context, postID uint64) ([]*model.PostMetric, error)
	GetPostMetricsBy30Days(ctx context.Context, postID uint64) ([]*model.PostMetric, error)
	GetLatestMetricBefore(ctx context.Context, postID uint64, date time.Time) (*model.PostMetric, error)
}

type postMetricRepoImpl struct {
	db *gorm.DB
}

func NewPostMetricRepository(db *gorm.DB) PostMetricRepo {
	return &postMetricRepoImpl{db: db}
}

// SaveOrUpdateMetric 采用 Upsert 逻辑。如果 post_id + metric_date 已存在，则更新各项数值
func (r *postMetricRepoImpl) SaveOrUpdateMetric(ctx context.Context, metric *model.PostMetric) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "post_id"}, {Name: "metric_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_likes",
			"total_comments",
			"total_collects",
			"total_views",
		}),
	}).Create(metric).Error
}

// GetPostMetricsBy7Days 获取帖子最近 7 天的趋势数据
func (r *postMetricRepoImpl) GetPostMetricsBy7Days(ctx context.Context, postID uint64) ([]*model.PostMetric, error) {
	metrics := make([]*model.PostMetric, 0)
	result := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -7)).
		Order("metric_date ASC").
		Find(&metrics)
	if result.Error != nil {
		return nil, result.Error
	}
	return metrics, nil
}

// GetPostMetricsBy30Days 获取帖子最近 30 天的趋势数据
func (r *postMetricRepoImpl) GetPostMetricsBy30Days(ctx context.Context, postID uint64) ([]*model.PostMetric, error) {
	metrics := make([]*model.PostMetric, 0)
	result := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -30)).
		Order("metric_date ASC").
		Find(&metrics)
	if result.Error != nil {
		return nil, result.Error
	}
	return metrics, nil
}

// GetLatestMetricBefore 获取指定日期前最近的一条指标记录（常用于计算增量）
func (r *postMetricRepoImpl) GetLatestMetricBefore(ctx context.Context, postID uint64, date time.Time) (*model.PostMetric, error) {
	var metric model.PostMetric
	err := r.db.WithContext(ctx).
		Where("post_id = ? AND metric_date < ?", postID, date).
		Order("metric_date DESC").
		First(&metric).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &metric, nil
}
