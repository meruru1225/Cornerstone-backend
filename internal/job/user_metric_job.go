package job

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/logger"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/service"
	"context"
	log "log/slog"
	"time"

	"github.com/google/uuid"
)

type UserMetricsJob struct {
	userSvc       service.UserService
	userMetricSvc service.UserMetricsService
	userFollowSvc service.UserFollowService
}

func NewUserMetricsJob(
	userSvc service.UserService,
	userMetricSvc service.UserMetricsService,
	userFollowSvc service.UserFollowService,
) *UserMetricsJob {
	return &UserMetricsJob{
		userSvc:       userSvc,
		userMetricSvc: userMetricSvc,
		userFollowSvc: userFollowSvc,
	}
}

func (s *UserMetricsJob) Run() {
	traceID := "job-" + uuid.NewString()
	ctx := context.WithValue(context.Background(), logger.TraceIDKey, traceID)

	processingKey := consts.UserFollowDirtyKey + ":processing"
	err := redis.Rename(ctx, consts.UserFollowDirtyKey, processingKey)
	if err != nil {
		return
	}

	tempSet, err := redis.GetSet(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "get dirty set error", "err", err)
		return
	}

	set, err := util.StrSliceToUInt64Slice(tempSet)
	if err != nil {
		log.ErrorContext(ctx, "convert set to int slice error", "err", err)
		return
	}

	for _, v := range set {
		err = s.userMetricSvc.SyncUserDailyMetric(ctx, v)
		if err != nil {
			log.ErrorContext(ctx, "create user metrics error", "err", err)
		}
		followerCount, err := s.userFollowSvc.GetUserFollowerCount(ctx, v)
		if err != nil {
			log.ErrorContext(ctx, "get user follower count error", "err", err)
		}
		followingCount, err := s.userFollowSvc.GetUserFollowingCount(ctx, v)
		if err != nil {
			log.ErrorContext(ctx, "get user following count error", "err", err)
		}
		err = s.userSvc.UpdateUserFollowCount(ctx, v, followerCount, followingCount)
		if err != nil {
			log.ErrorContext(ctx, "update user follow count error", "err", err)
		}
	}

	err = redis.DeleteKey(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "delete dirty set error", "err", err)
	}

	log.InfoContext(ctx, "sync user metrics success", "date", time.DateOnly)
}
