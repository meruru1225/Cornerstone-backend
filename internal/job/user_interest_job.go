package job

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/logger"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type UserInterestJob struct {
	interestRepo repository.UserInterestRepo
}

func NewUserInterestJob(interestRepo repository.UserInterestRepo) *UserInterestJob {
	return &UserInterestJob{
		interestRepo: interestRepo,
	}
}

func (s *UserInterestJob) Run() {
	traceID := "job-interest-" + uuid.NewString()
	ctx := context.WithValue(context.Background(), logger.TraceIDKey, traceID)

	processingKey := consts.UserInterestDirtyKey + ":processing"
	err := redis.Rename(ctx, consts.UserInterestDirtyKey, processingKey)
	if err != nil {
		return
	}

	tempSet, err := redis.GetSet(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "get interest dirty set error", "err", err)
		return
	}

	userIDs, err := util.StrSliceToUInt64Slice(tempSet)
	if err != nil {
		log.ErrorContext(ctx, "convert interest set to int slice error", "err", err)
		return
	}

	log.InfoContext(ctx, "UserInterestJob processing", "user_count", len(userIDs))

	for _, uid := range userIDs {
		uidStr := strconv.FormatUint(uid, 10)
		interestKey := consts.UserInterestKey + uidStr

		zObjects, err := redis.ZRevRangeWithScores(ctx, interestKey, 0, 100)
		if err != nil {
			log.ErrorContext(ctx, "fetch zset error", "uid", uid, "err", err)
			continue
		}

		if len(zObjects) == 0 {
			continue
		}

		interestMap := make(model.InterestMap)
		for _, obj := range zObjects {
			if tag, ok := obj.Member.(string); ok {
				interestMap[tag] = int64(obj.Score)
			}
		}

		metric := &model.UserInterestTags{
			UserID:    uid,
			Interests: interestMap,
			UpdatedAt: time.Now(),
		}

		err = s.interestRepo.SaveUserInterests(ctx, metric)
		if err != nil {
			log.ErrorContext(ctx, "save user interests to mysql error", "uid", uid, "err", err)
			continue
		}
	}

	err = redis.DeleteKey(ctx, processingKey)
	if err != nil {
		log.ErrorContext(ctx, "delete interest processing set error", "err", err)
	}

	log.InfoContext(ctx, "UserInterestJob finished", "processed_count", len(userIDs))
}
