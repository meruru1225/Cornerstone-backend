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

type PostCommentJob struct {
	actionSvc service.PostActionService
}

func NewPostCommentJob(actionSvc service.PostActionService) *PostCommentJob {
	return &PostCommentJob{
		actionSvc: actionSvc,
	}
}

func (s *PostCommentJob) Run() {
	traceID := "job-comment-" + uuid.NewString()
	ctx := context.WithValue(context.Background(), logger.TraceIDKey, traceID)

	processingKey := consts.PostCommentLikeDirtyKey + ":processing"
	err := redis.Rename(ctx, consts.PostCommentLikeDirtyKey, processingKey)
	if err != nil {
		return
	}

	tempSet, err := redis.GetSet(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "get comment dirty set error", "err", err)
		return
	}

	commentIDs, err := util.StrSliceToUInt64Slice(tempSet)
	if err != nil {
		log.ErrorContext(ctx, "convert comment set to int slice error", "err", err)
		return
	}

	log.InfoContext(ctx, "start syncing comment likes count", "count", len(commentIDs))

	successCount := 0
	for _, cid := range commentIDs {
		count, err := s.actionSvc.GetCommentLikeCount(ctx, cid)
		if err != nil {
			log.ErrorContext(ctx, "get comment like count error", "cid", cid, "err", err)
			continue
		}

		err = s.actionSvc.SyncCommentLikesCount(ctx, cid, int(count))
		if err != nil {
			log.ErrorContext(ctx, "sync comment likes count to mysql error", "cid", cid, "err", err)
			continue
		}
		successCount++
	}

	err = redis.DeleteKey(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "delete comment processing set error", "err", err)
	}

	log.InfoContext(ctx, "sync comment metrics success",
		"total_count", len(commentIDs),
		"success_count", successCount)
}
