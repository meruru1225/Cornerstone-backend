package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserContentMetricRepo interface {
	SaveOrUpdateMetric(ctx context.Context, metric *model.UserContentMetric) error
	GetUserContentMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserContentMetric, error)
	GetUserContentMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserContentMetric, error)
	GetLatestMetricBefore(ctx context.Context, userID uint64, date time.Time) (*model.UserContentMetric, error)
}

type userContentMetricRepoImpl struct {
	db *gorm.DB
}

func NewUserContentMetricRepository(db *gorm.DB) UserContentMetricRepo {
	return &userContentMetricRepoImpl{db: db}
}

func (r *userContentMetricRepoImpl) SaveOrUpdateMetric(ctx context.Context, metric *model.UserContentMetric) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "metric_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_likes",
			"total_collects",
			"total_comments",
			"total_views",
		}),
	}).Create(metric).Error
}

func (r *userContentMetricRepoImpl) GetUserContentMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserContentMetric, error) {
	metrics := make([]*model.UserContentMetric, 0)
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -7)).
		Order("metric_date ASC").
		Find(&metrics)
	return metrics, result.Error
}

func (r *userContentMetricRepoImpl) GetUserContentMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserContentMetric, error) {
	metrics := make([]*model.UserContentMetric, 0)
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -30)).
		Order("metric_date ASC").
		Find(&metrics)
	return metrics, result.Error
}

func (r *userContentMetricRepoImpl) GetLatestMetricBefore(ctx context.Context, userID uint64, date time.Time) (*model.UserContentMetric, error) {
	var metric model.UserContentMetric
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND metric_date < ?", userID, date).
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
