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

type PostMetricService interface {
	// SyncPostMetric 同步帖子每日指标快照
	SyncPostMetric(ctx context.Context, postID uint64) error
	// GetPostMetricsBy7Days 获取最近7天全维度趋势数据
	GetPostMetricsBy7Days(ctx context.Context, postID uint64, userID uint64) (*dto.PostTrendDTO, error)
	// GetPostMetricsBy30Days 获取最近30天全维度趋势数据
	GetPostMetricsBy30Days(ctx context.Context, postID uint64, userID uint64) (*dto.PostTrendDTO, error)
}

type postMetricServiceImpl struct {
	postMetricRepo repository.PostMetricRepo
	postRepo       repository.PostRepo
}

func NewPostMetricService(postMetricRepo repository.PostMetricRepo, postRepo repository.PostRepo) PostMetricService {
	return &postMetricServiceImpl{
		postMetricRepo: postMetricRepo,
		postRepo:       postRepo,
	}
}

// SyncPostMetric 实现：将 posts 表的实时计数刷入每日指标表
func (s *postMetricServiceImpl) SyncPostMetric(ctx context.Context, postID uint64) error {
	post, err := s.postRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}

	today := util.GetMidnight(time.Now())
	metric := &model.PostMetric{
		PostID:        postID,
		MetricDate:    today,
		TotalLikes:    post.LikesCount,
		TotalComments: post.CommentsCount,
		TotalCollects: post.CollectsCount,
		TotalViews:    post.ViewsCount,
	}

	err = s.postMetricRepo.SaveOrUpdateMetric(ctx, metric)
	if err != nil {
		return err
	}

	_ = redis.DeleteKey(ctx, consts.PostMetrics7DaysKey+strconv.FormatUint(postID, 10))
	_ = redis.DeleteKey(ctx, consts.PostMetrics30DaysKey+strconv.FormatUint(postID, 10))

	return nil
}

func (s *postMetricServiceImpl) GetPostMetricsBy7Days(ctx context.Context, postID uint64, userID uint64) (*dto.PostTrendDTO, error) {
	key := consts.PostMetrics7DaysKey + strconv.FormatUint(postID, 10)
	return s.getPostMetrics(ctx, postID, userID, key, 7, func() ([]*model.PostMetric, error) {
		return s.postMetricRepo.GetPostMetricsBy7Days(ctx, postID)
	})
}

func (s *postMetricServiceImpl) GetPostMetricsBy30Days(ctx context.Context, postID uint64, userID uint64) (*dto.PostTrendDTO, error) {
	key := consts.PostMetrics30DaysKey + strconv.FormatUint(postID, 10)
	return s.getPostMetrics(ctx, postID, userID, key, 30, func() ([]*model.PostMetric, error) {
		return s.postMetricRepo.GetPostMetricsBy30Days(ctx, postID)
	})
}

// getPostMetrics 聚合查询与数据平滑逻辑
func (s *postMetricServiceImpl) getPostMetrics(
	ctx context.Context,
	postID uint64,
	userID uint64,
	key string,
	days int,
	fetchDB func() ([]*model.PostMetric, error),
) (*dto.PostTrendDTO, error) {
	post, err := s.postRepo.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}
	if post.UserID != userID {
		return nil, UnauthorizedError
	}

	if val, err := redis.GetValue(ctx, key); err == nil && val != "" {
		var res dto.PostTrendDTO
		_ = json.Unmarshal([]byte(val), &res)
		return &res, nil
	}

	rawData, err := fetchDB()
	if err != nil {
		return nil, err
	}

	startTime := util.GetMidnight(time.Now()).AddDate(0, 0, -days)
	var baseline *model.PostMetric
	if len(rawData) == 0 || !rawData[0].MetricDate.Equal(startTime) {
		baseline, _ = s.postMetricRepo.GetLatestMetricBefore(ctx, postID, startTime)
	} else {
		baseline = rawData[0]
	}

	dataMap := make(map[string]*model.PostMetric)
	for _, m := range rawData {
		dataMap[m.MetricDate.Format(time.DateOnly)] = m
	}

	res := &dto.PostTrendDTO{
		PostID:   postID,
		Days:     days,
		Likes:    make([]*dto.PostMetricDTO, 0, days),
		Comments: make([]*dto.PostMetricDTO, 0, days),
		Collects: make([]*dto.PostMetricDTO, 0, days),
		Views:    make([]*dto.PostMetricDTO, 0, days),
	}

	var lastValid = baseline
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		currentDate := util.GetMidnight(now.AddDate(0, 0, -i))
		dateStr := currentDate.Format(time.DateOnly)

		l, c, col, v := 0, 0, 0, 0
		if val, ok := dataMap[dateStr]; ok {
			l, c, col, v = val.TotalLikes, val.TotalComments, val.TotalCollects, val.TotalViews
			lastValid = val
		} else if lastValid != nil {
			l, c, col, v = lastValid.TotalLikes, lastValid.TotalComments, lastValid.TotalCollects, lastValid.TotalViews
		}

		res.Likes = append(res.Likes, &dto.PostMetricDTO{Date: dateStr, Value: l})
		res.Comments = append(res.Comments, &dto.PostMetricDTO{Date: dateStr, Value: c})
		res.Collects = append(res.Collects, &dto.PostMetricDTO{Date: dateStr, Value: col})
		res.Views = append(res.Views, &dto.PostMetricDTO{Date: dateStr, Value: v})
	}

	_ = redis.SetWithMidnightExpiration(ctx, key, res)

	return res, nil
}
