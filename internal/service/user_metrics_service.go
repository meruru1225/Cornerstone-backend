package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
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

	today := getMidnight(time.Now())
	metric := &model.UserMetrics{
		UserID:         userID,
		TotalFollowers: int(followerCount),
		MetricDate:     today,
	}

	return s.userMetricsRepo.SaveOrUpdateMetric(ctx, metric)
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
	list, err := redis.GetList(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(list) != 0 {
		metrics := make([]*dto.UserMetricDTO, 0, len(list))
		for _, v := range list {
			var metric *dto.UserMetricDTO
			if err := json.Unmarshal([]byte(v), &metric); err != nil {
				return nil, err
			}
			metrics = append(metrics, metric)
		}
		return metrics, nil
	}

	rawData, err := fetchFromDB()
	if err != nil {
		return nil, err
	}

	startTime := getMidnight(time.Now()).AddDate(0, 0, -days)

	var baseline *model.UserMetrics
	if len(rawData) == 0 || !rawData[0].MetricDate.Equal(startTime) {
		baseline, _ = s.userMetricsRepo.GetLatestMetricBefore(ctx, userID, startTime)
	} else {
		baseline = rawData[0]
	}

	finalMetrics := s.fillMetricsGaps(rawData, days, baseline)

	s.cacheMetrics(ctx, key, finalMetrics)
	return finalMetrics, nil
}

func (s *userMetricsServiceImpl) cacheMetrics(ctx context.Context, key string, metrics []*dto.UserMetricDTO) {
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

func (s *userMetricsServiceImpl) fillMetricsGaps(rawData []*model.UserMetrics, days int, baseline *model.UserMetrics) []*dto.UserMetricDTO {
	dataMap := make(map[string]*model.UserMetrics)
	for _, m := range rawData {
		dataMap[m.MetricDate.Format(time.DateOnly)] = m
	}

	finalDTOs := make([]*dto.UserMetricDTO, 0, days)
	now := time.Now()
	var lastValid = baseline

	for i := days - 1; i >= 0; i-- {
		currentDate := getMidnight(now.AddDate(0, 0, -i))
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

func getMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
