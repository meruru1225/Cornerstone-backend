package repository

import (
	"Cornerstone/internal/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserMetricsRepo interface {
	SaveOrUpdateMetric(ctx context.Context, metric *model.UserMetrics) error
	GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error)
	GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error)
	GetLatestMetricBefore(ctx context.Context, userID uint64, date time.Time) (*model.UserMetrics, error)
}

type userMetricsRepoImpl struct {
	db *gorm.DB
}

func NewUserMetricsRepository(db *gorm.DB) UserMetricsRepo {
	return &userMetricsRepoImpl{db: db}
}

func (s *userMetricsRepoImpl) SaveOrUpdateMetric(ctx context.Context, metric *model.UserMetrics) error {
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "metric_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_followers", "updated_at"}),
	}).Create(metric).Error
}

func (s *userMetricsRepoImpl) GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error) {
	metrics := make([]*model.UserMetrics, 0)
	result := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -7)).
		Find(&metrics)
	if result.Error != nil {
		return nil, result.Error
	}
	return metrics, nil
}

func (s *userMetricsRepoImpl) GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error) {
	metrics := make([]*model.UserMetrics, 0)
	result := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("metric_date >= ?", time.Now().AddDate(0, 0, -30)).
		Find(&metrics)
	if result.Error != nil {
		return nil, result.Error
	}
	return metrics, nil
}

func (s *userMetricsRepoImpl) GetLatestMetricBefore(ctx context.Context, userID uint64, date time.Time) (*model.UserMetrics, error) {
	var metric model.UserMetrics
	// 找指定日期之前最近的一条记录
	err := s.db.WithContext(ctx).
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
