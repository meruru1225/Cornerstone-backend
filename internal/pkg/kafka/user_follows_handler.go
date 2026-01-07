package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"context"
	log "log/slog"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	redisv9 "github.com/redis/go-redis/v9"
)

type UserFollowsHandler struct {
}

func NewUserFollowsConsumer() *UserFollowsHandler {
	return &UserFollowsHandler{}
}

func (s *UserFollowsHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("user follows consumer setup")
	return nil
}

func (s *UserFollowsHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("user follows consumer cleanup")
	return nil
}

func (s *UserFollowsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-user-follows consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-user-follows process batch error", "err", err)
		return err
	}
	log.Info("topic-user-follows consume claim end")
	return nil
}

func (s *UserFollowsHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	canalMsg, err := ToCanalMessage(msg, "user_follows")
	if err != nil || canalMsg == nil {
		return nil
	}

	rdb := redis.GetRdbClient()

	pipe := rdb.Pipeline()
	var affectedUIDs []interface{}

	for _, row := range canalMsg.Data {
		followerID := StrToUint64(row["follower_id"])
		followingID := StrToUint64(row["following_id"])
		affectedUIDs = append(affectedUIDs, followerID, followingID)

		fdrKey := consts.UserFollowerKey + strconv.FormatUint(followingID, 10)
		fngKey := consts.UserFollowingKey + strconv.FormatUint(followerID, 10)
		fdrCountKey := consts.UserFollowerCountKey + strconv.FormatUint(followingID, 10)
		fngCountKey := consts.UserFollowingCountKey + strconv.FormatUint(followerID, 10)

		if canalMsg.Type == INSERT {
			now := float64(time.Now().Unix())
			pipe.ZAdd(ctx, fdrKey, redisv9.Z{Score: now, Member: followerID})
			pipe.ZRemRangeByRank(ctx, fdrKey, 0, -1001)
			pipe.ZAdd(ctx, fngKey, redisv9.Z{Score: now, Member: followingID})
			pipe.ZRemRangeByRank(ctx, fngKey, 0, -1001)
			pipe.Incr(ctx, fdrCountKey)
			pipe.Incr(ctx, fngCountKey)
		} else if canalMsg.Type == DELETE {
			pipe.ZRem(ctx, fdrKey, followerID)
			pipe.ZRem(ctx, fngKey, followingID)
			pipe.Decr(ctx, fdrCountKey)
			pipe.Decr(ctx, fngCountKey)
		}
	}

	if len(affectedUIDs) > 0 {
		pipe.SAdd(ctx, consts.UserFollowDirtyKey, affectedUIDs...)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Error("Redis Pipeline Exec failed", "err", err, "msg_key", string(msg.Key))
		return err
	}

	return nil
}
