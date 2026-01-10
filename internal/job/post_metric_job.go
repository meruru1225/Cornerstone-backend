package job

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/logger"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/service"
	"context"
	log "log/slog"

	"github.com/google/uuid"
)

type PostMetricsJob struct {
	postSvc                  service.PostService
	postMetricSvc            service.PostMetricService
	actionSvc                service.PostActionService
	userContentMetricService service.UserContentMetricService
}

func NewPostMetricsJob(
	postSvc service.PostService,
	postMetricSvc service.PostMetricService,
	actionSvc service.PostActionService,
	userContentMetricSvc service.UserContentMetricService,
) *PostMetricsJob {
	return &PostMetricsJob{
		postSvc:                  postSvc,
		postMetricSvc:            postMetricSvc,
		actionSvc:                actionSvc,
		userContentMetricService: userContentMetricSvc,
	}
}

func (s *PostMetricsJob) Run() {
	traceID := "job-post-" + uuid.NewString()
	ctx := context.WithValue(context.Background(), logger.TraceIDKey, traceID)

	processingKey := consts.PostDirtyKey + ":processing"
	err := redis.Rename(ctx, consts.PostDirtyKey, processingKey)
	if err != nil {
		return
	}

	tempSet, err := redis.GetSet(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "get post dirty set error", "err", err)
		return
	}

	postIDs, err := util.StrSliceToUInt64Slice(tempSet)
	if err != nil {
		log.ErrorContext(ctx, "convert post set to int slice error", "err", err)
		return
	}

	dirtyUserIDs := make(map[uint64]struct{})

	for _, pid := range postIDs {
		likes, _ := s.actionSvc.GetPostLikeCount(ctx, pid)
		comments, _ := s.actionSvc.GetPostCommentCount(ctx, pid)
		collects, _ := s.actionSvc.GetPostCollectionCount(ctx, pid)
		views, _ := s.actionSvc.GetPostViewCount(ctx, pid)

		err = s.postSvc.UpdatePostCounts(ctx, pid, likes, comments, collects, views)
		if err != nil {
			log.ErrorContext(ctx, "update post counts error", "pid", pid, "err", err)
			continue
		}

		err = s.postMetricSvc.SyncPostMetric(ctx, pid)
		if err != nil {
			log.ErrorContext(ctx, "sync post daily metric error", "pid", pid, "err", err)
		}

		post, err := s.postSvc.GetPostById(ctx, pid)
		if err == nil && post != nil {
			dirtyUserIDs[post.UserID] = struct{}{}
		}
	}

	for uid := range dirtyUserIDs {
		err := s.userContentMetricService.SyncUserContentMetric(ctx, uid)
		if err != nil {
			log.ErrorContext(ctx, "sync user content metric error", "uid", uid, "err", err)
		}
	}

	err = redis.DeleteKey(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "delete post processing set error", "err", err)
	}

	log.InfoContext(ctx, "sync post metrics success",
		"post_count", len(postIDs),
		"user_count", len(dirtyUserIDs))
}
