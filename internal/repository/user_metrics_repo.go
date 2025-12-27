package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type UserMetricsRepo interface {
	CreateUserMetric(ctx context.Context, metric *model.UserMetrics) error
	UpdateUserMetric(ctx context.Context, metric *model.UserMetrics) error
	GetUserMetricsByDate(ctx context.Context, userID uint64, date time.Time) (*model.UserMetrics, error)
	GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error)
	GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error)
}

type userMetricsRepoImpl struct {
	db *gorm.DB
}

func NewUserMetricsRepository(db *gorm.DB) UserMetricsRepo {
	return &userMetricsRepoImpl{db: db}
}

func (r *userMetricsRepoImpl) CreateUserMetric(ctx context.Context, metric *model.UserMetrics) error {
	return r.db.WithContext(ctx).Create(metric).Error
}

func (r *userMetricsRepoImpl) UpdateUserMetric(ctx context.Context, metric *model.UserMetrics) error {
	return r.db.WithContext(ctx).
		Select("total_followers").
		Updates(metric).Error
}

func (r *userMetricsRepoImpl) GetUserMetricsByDate(ctx context.Context, userID uint64, date time.Time) (*model.UserMetrics, error) {
	var metric model.UserMetrics
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("metric_date = ?", date).
		First(&metric)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &metric, nil
}

func (r *userMetricsRepoImpl) GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error) {
	metrics := make([]*model.UserMetrics, 0)
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -7)).
		Find(&metrics)
	if result.Error != nil {
		return nil, result.Error
	}
	return metrics, nil
}

func (r *userMetricsRepoImpl) GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error) {
	metrics := make([]*model.UserMetrics, 0)
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -30)).
		Find(&metrics)
	if result.Error != nil {
		return nil, result.Error
	}
	return metrics, nil
}
