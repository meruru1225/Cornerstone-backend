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

type UserContentMetricService interface {
	// SyncUserContentMetric 聚合该用户所有帖子的数据并记录快照
	SyncUserContentMetric(ctx context.Context, userID uint64) error
	// GetUserContentMetricsBy7Days 获取最近7天创作者表现趋势
	GetUserContentMetricsBy7Days(ctx context.Context, userID uint64) (*dto.UserContentTrendDTO, error)
	// GetUserContentMetricsBy30Days 获取最近30天创作者表现趋势
	GetUserContentMetricsBy30Days(ctx context.Context, userID uint64) (*dto.UserContentTrendDTO, error)
}

type userContentServiceImpl struct {
	userContentMetricRepo repository.UserContentMetricRepo
	postRepo              repository.PostRepo
	actionRepo            repository.PostActionRepo
}

func NewUserContentMetricService(
	userContentMetricRepo repository.UserContentMetricRepo,
	postRepo repository.PostRepo,
	actionRepo repository.PostActionRepo,
) UserContentMetricService {
	return &userContentServiceImpl{
		userContentMetricRepo: userContentMetricRepo,
		postRepo:              postRepo,
		actionRepo:            actionRepo,
	}
}

// SyncUserContentMetric 从 posts 表聚合该用户所有作品的当前总数并存入快照
func (s *userContentServiceImpl) SyncUserContentMetric(ctx context.Context, userID uint64) error {
	likes, err := s.actionRepo.GetUserTotalLikes(ctx, userID)
	if err != nil {
		return err
	}

	comments, err := s.actionRepo.GetUserTotalComments(ctx, userID)
	if err != nil {
		return err
	}

	collects, err := s.actionRepo.GetUserTotalCollects(ctx, userID) // 补齐
	if err != nil {
		return err
	}

	views, err := s.actionRepo.GetUserTotalViews(ctx, userID)
	if err != nil {
		return err
	}

	today := util.GetMidnight(time.Now())
	metric := &model.UserContentMetric{
		UserID:        userID,
		MetricDate:    today,
		TotalLikes:    int(likes),
		TotalComments: int(comments),
		TotalCollects: int(collects),
		TotalViews:    int(views),
	}

	return s.userContentMetricRepo.SaveOrUpdateMetric(ctx, metric)
}

func (s *userContentServiceImpl) GetUserContentMetricsBy7Days(ctx context.Context, userID uint64) (*dto.UserContentTrendDTO, error) {
	key := consts.UserContentMetrics7DaysKey + strconv.FormatUint(userID, 10)
	return s.getUserContentMetrics(ctx, userID, key, 7, func() ([]*model.UserContentMetric, error) {
		return s.userContentMetricRepo.GetUserContentMetricsBy7Days(ctx, userID)
	})
}

func (s *userContentServiceImpl) GetUserContentMetricsBy30Days(ctx context.Context, userID uint64) (*dto.UserContentTrendDTO, error) {
	key := consts.UserContentMetrics30DaysKey + strconv.FormatUint(userID, 10)
	return s.getUserContentMetrics(ctx, userID, key, 30, func() ([]*model.UserContentMetric, error) {
		return s.userContentMetricRepo.GetUserContentMetricsBy30Days(ctx, userID)
	})
}

// getUserContentMetrics 核心补全与平滑逻辑
func (s *userContentServiceImpl) getUserContentMetrics(
	ctx context.Context,
	userID uint64,
	key string,
	days int,
	fetchDB func() ([]*model.UserContentMetric, error),
) (*dto.UserContentTrendDTO, error) {
	if val, err := redis.GetValue(ctx, key); err == nil && val != "" {
		var res dto.UserContentTrendDTO
		_ = json.Unmarshal([]byte(val), &res)
		return &res, nil
	}

	rawData, err := fetchDB()
	if err != nil {
		return nil, err
	}

	startTime := util.GetMidnight(time.Now()).AddDate(0, 0, -days)
	var baseline *model.UserContentMetric
	if len(rawData) == 0 || !rawData[0].MetricDate.Equal(startTime) {
		baseline, _ = s.userContentMetricRepo.GetLatestMetricBefore(ctx, userID, startTime)
	} else {
		baseline = rawData[0]
	}

	dataMap := make(map[string]*model.UserContentMetric)
	for _, m := range rawData {
		dataMap[m.MetricDate.Format(time.DateOnly)] = m
	}

	res := &dto.UserContentTrendDTO{
		UserID:   userID,
		Days:     days,
		Likes:    make([]*dto.PostMetricDTO, 0, days),
		Comments: make([]*dto.PostMetricDTO, 0, days),
		Views:    make([]*dto.PostMetricDTO, 0, days),
	}

	var lastValid = baseline
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		currentDate := util.GetMidnight(now.AddDate(0, 0, -i))
		dateStr := currentDate.Format(time.DateOnly)

		l, c, v := 0, 0, 0
		if val, ok := dataMap[dateStr]; ok {
			l, c, v = val.TotalLikes, val.TotalComments, val.TotalViews
			lastValid = val
		} else if lastValid != nil {
			// 如果当天没有生成快照记录，则继承最近一次有效的累计值
			l, c, v = lastValid.TotalLikes, lastValid.TotalComments, lastValid.TotalViews
		}

		res.Likes = append(res.Likes, &dto.PostMetricDTO{Date: dateStr, Value: l})
		res.Comments = append(res.Comments, &dto.PostMetricDTO{Date: dateStr, Value: c})
		res.Views = append(res.Views, &dto.PostMetricDTO{Date: dateStr, Value: v})
	}

	_ = redis.SetWithMidnightExpiration(ctx, key, res)

	return res, nil
}
