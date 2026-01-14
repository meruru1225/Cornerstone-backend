package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/repository"
	"context"
	"strconv"
	"time"

	"github.com/goccy/go-json"
)

type UserMetricsService interface {
	SyncUserDailyMetric(ctx context.Context, userID uint64) error
	GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*dto.UserMetricDTO, error)
	GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*dto.UserMetricDTO, error)
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

func (s *userMetricsServiceImpl) SyncUserDailyMetric(ctx context.Context, userID uint64) error {
	followerCount, err := s.userFollowRepo.GetUserFollowerCount(ctx, userID)
	if err != nil {
		return err
	}

	today := util.GetMidnight(time.Now())
	metric := &model.UserMetrics{
		UserID:         userID,
		TotalFollowers: int(followerCount),
		MetricDate:     today,
	}

	err = s.userMetricsRepo.SaveOrUpdateMetric(ctx, metric)
	if err != nil {
		return err
	}

	_ = redis.DeleteKey(ctx, consts.UserMetrics7DaysKey+strconv.FormatUint(userID, 10))
	_ = redis.DeleteKey(ctx, consts.UserMetrics30DaysKey+strconv.FormatUint(userID, 10))
	return nil
}

func (s *userMetricsServiceImpl) GetUserMetricsBy7Days(ctx context.Context, userID uint64) ([]*dto.UserMetricDTO, error) {
	key := consts.UserMetrics7DaysKey + strconv.FormatUint(userID, 10)
	return s.getUserMetricsByDays(ctx, userID, key, 7, func() ([]*model.UserMetrics, error) {
		return s.userMetricsRepo.GetUserMetricsBy7Days(ctx, userID)
	})
}

func (s *userMetricsServiceImpl) GetUserMetricsBy30Days(ctx context.Context, userID uint64) ([]*dto.UserMetricDTO, error) {
	key := consts.UserMetrics30DaysKey + strconv.FormatUint(userID, 10)
	return s.getUserMetricsByDays(ctx, userID, key, 30, func() ([]*model.UserMetrics, error) {
		return s.userMetricsRepo.GetUserMetricsBy30Days(ctx, userID)
	})
}

func (s *userMetricsServiceImpl) getUserMetricsByDays(
	ctx context.Context,
	userID uint64,
	key string,
	days int,
	fetchFromDB func() ([]*model.UserMetrics, error),
) ([]*dto.UserMetricDTO, error) {
	if val, err := redis.GetValue(ctx, key); err == nil && val != "" {
		var metrics []*dto.UserMetricDTO
		if err := json.Unmarshal([]byte(val), &metrics); err == nil {
			return metrics, nil
		}
	}

	rawData, err := fetchFromDB()
	if err != nil {
		return nil, err
	}

	startTime := util.GetMidnight(time.Now()).AddDate(0, 0, -days)
	var baseline *model.UserMetrics
	if len(rawData) == 0 || !rawData[0].MetricDate.Equal(startTime) {
		baseline, _ = s.userMetricsRepo.GetLatestMetricBefore(ctx, userID, startTime)
	} else {
		baseline = rawData[0]
	}

	finalMetrics := s.fillMetricsGaps(rawData, days, baseline)

	_ = redis.SetWithMidnightExpiration(ctx, key, finalMetrics)

	return finalMetrics, nil
}

func (s *userMetricsServiceImpl) fillMetricsGaps(rawData []*model.UserMetrics, days int, baseline *model.UserMetrics) []*dto.UserMetricDTO {
	dataMap := make(map[string]*model.UserMetrics)
	for _, m := range rawData {
		dataMap[m.MetricDate.Format(time.DateOnly)] = m
	}

	finalDTOs := make([]*dto.UserMetricDTO, 0, days)
	now := time.Now()
	var lastValid = baseline

	for i := days - 1; i >= 0; i-- {
		currentDate := util.GetMidnight(now.AddDate(0, 0, -i))
		dateStr := currentDate.Format(time.DateOnly)

		count := 0
		if val, ok := dataMap[dateStr]; ok {
			count = val.TotalFollowers
			lastValid = val
		} else if lastValid != nil {
			count = lastValid.TotalFollowers
		}

		finalDTOs = append(finalDTOs, &dto.UserMetricDTO{
			Date:  dateStr,
			Value: count,
		})
	}

	return finalDTOs
}
