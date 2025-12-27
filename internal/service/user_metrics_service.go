package service

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

type UserMetricsService interface {
	CreateUserMetricByYesterday(ctx context.Context, userID uint64) error
	AddCountUserMetrics(ctx context.Context, userID uint64, count int) error
	GetUserMetricsByDate(ctx context.Context, userID uint64, date time.Time) (*model.UserMetrics, error)
	GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error)
	GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error)
}

type userMetricsServiceImpl struct {
	userMetricsRepo repository.UserMetricsRepo
	userFollowRepo  repository.UserFollowRepo
}

func NewUserMetricsService(userMetricsRepo repository.UserMetricsRepo, userFollowRepo repository.UserFollowRepo) UserMetricsService {
	return &userMetricsServiceImpl{
		userMetricsRepo: userMetricsRepo,
		userFollowRepo:  userFollowRepo,
	}
}

func (s *userMetricsServiceImpl) CreateUserMetricByYesterday(ctx context.Context, userID uint64) error {
	lockKey := consts.UserMetricDailyLock + strconv.FormatUint(userID, 10)
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	lock, err := redis.TryLock(ctx, lockKey, newUUID.String(), time.Minute*5, 3)
	if err != nil {
		return err
	}
	if !lock {
		return UnExpectedError
	}
	defer redis.UnLock(ctx, lockKey, newUUID.String())
	yesterday := getMidnight(time.Now()).AddDate(0, 0, -1)
	metric, err := s.userMetricsRepo.GetUserMetricsByDate(ctx, userID, yesterday)
	if err != nil {
		return err
	}
	if metric == nil {
		followerCount, err := s.userFollowRepo.GetUserFollowerCount(ctx, userID)
		if err != nil {
			return err
		}
		metric = &model.UserMetrics{
			UserID:         userID,
			TotalFollowers: int(followerCount),
		}
	}
	metric.ID = 0
	metric.MetricDate = getMidnight(time.Now())
	return s.userMetricsRepo.CreateUserMetric(ctx, metric)
}

func (s *userMetricsServiceImpl) AddCountUserMetrics(ctx context.Context, userID uint64, count int) error {
	lockKey := consts.UserMetricDailyLock + strconv.FormatUint(userID, 10)
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	lock, err := redis.TryLock(ctx, lockKey, newUUID.String(), time.Minute*5, 3)
	if err != nil {
		return UnExpectedError
	}
	if !lock {
		return nil
	}
	defer redis.UnLock(ctx, lockKey, newUUID.String())
	now := getMidnight(time.Now())
	metric, err := s.userMetricsRepo.GetUserMetricsByDate(ctx, userID, now)
	if err != nil {
		return err
	}
	if metric == nil {
		yesterday := getMidnight(time.Now()).AddDate(0, 0, -1)
		metric, err = s.userMetricsRepo.GetUserMetricsByDate(ctx, userID, yesterday)
		if err != nil {
			return err
		}
		if metric == nil {
			followerCount, err := s.userFollowRepo.GetUserFollowerCount(ctx, userID)
			if err != nil {
				return err
			}
			metric = &model.UserMetrics{
				UserID:         userID,
				MetricDate:     now,
				TotalFollowers: int(followerCount),
			}
		} else {
			metric.ID = 0
			metric.MetricDate = getMidnight(time.Now())
			metric.TotalFollowers += count
		}
		return s.userMetricsRepo.CreateUserMetric(ctx, metric)
	}
	metric.TotalFollowers += count
	return s.userMetricsRepo.UpdateUserMetric(ctx, metric)
}

func (s *userMetricsServiceImpl) GetUserMetricsByDate(ctx context.Context, userID uint64, date time.Time) (*model.UserMetrics, error) {
	return s.userMetricsRepo.GetUserMetricsByDate(ctx, userID, date)
}

func (s *userMetricsServiceImpl) GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error) {
	key := consts.UserMetrics7DaysKey + strconv.FormatUint(userID, 10)
	return s.getUserMetricsByDays(ctx, key, func() ([]*model.UserMetrics, error) {
		return s.userMetricsRepo.GetUserMetricsBy7Days(ctx, userID)
	})
}

func (s *userMetricsServiceImpl) GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*model.UserMetrics, error) {
	key := consts.UserMetrics30DaysKey + strconv.FormatUint(userID, 10)
	return s.getUserMetricsByDays(ctx, key, func() ([]*model.UserMetrics, error) {
		return s.userMetricsRepo.GetUserMetricsBy30Days(ctx, userID)
	})
}

func (s *userMetricsServiceImpl) getUserMetricsByDays(
	ctx context.Context,
	key string,
	fetchFromDB func() ([]*model.UserMetrics, error),
) ([]*model.UserMetrics, error) {
	list, err := redis.GetList(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(list) != 0 {
		metrics := make([]*model.UserMetrics, 0, len(list))
		for _, v := range list {
			var metric *model.UserMetrics
			if err := json.Unmarshal([]byte(v), &metric); err != nil {
				return nil, err
			}
			metrics = append(metrics, metric)
		}
		return metrics, nil
	}

	metrics, err := fetchFromDB()
	if err != nil {
		return nil, err
	}

	s.cacheMetrics(ctx, key, metrics)
	return metrics, nil
}

func (s *userMetricsServiceImpl) cacheMetrics(ctx context.Context, key string, metrics []*model.UserMetrics) {
	metricJsons := make([]string, 0, len(metrics))
	for _, v := range metrics {
		metricJson, err := json.Marshal(v)
		if err != nil {
			return
		}
		metricJsons = append(metricJsons, string(metricJson))
	}

	// 计算距离午夜的时间，提前5分钟过期
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	expiration := time.Until(midnight) - time.Minute*5
	if expiration < 0 {
		return
	}

	_ = redis.SetListWithExpiration(ctx, key, metricJsons, expiration)
}

func getMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
