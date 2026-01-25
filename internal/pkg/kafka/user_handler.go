package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/redis"
	"context"
	"errors"
	log "log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type UserHandler struct {
	userESRepo es.UserRepo
}

func NewUserHandler(userESRepo es.UserRepo) *UserHandler {
	return &UserHandler{
		userESRepo: userESRepo,
	}
}

func (s *UserHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("user consumer setup")
	return nil
}

func (s *UserHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("user consumer cleanup")
	return nil
}

func (s *UserHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-user consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("process batch error", "err", err)
		return err
	}
	log.Info("topic-user consume claim end")
	return nil
}

func (s *UserHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	canalMsg, err := ToCanalMessage(msg, "users")
	if err != nil {
		return err
	}
	if len(canalMsg.Data) == 0 {
		return errors.New("canal message data is empty")
	}

	if canalMsg.Data[0]["is_delete"] == "1" {
		id := canalMsg.Data[0]["id"]
		lockKey := consts.UserDetailESLock + id.(string)
		uuidStr := uuid.NewString()
		_, err = redis.TryLock(ctx, lockKey, uuidStr, 30*time.Second, -1)
		defer redis.UnLock(ctx, lockKey, uuidStr)
		if err != nil {
			return err
		}
		return s.userESRepo.DeleteUser(ctx, StrToUint64(canalMsg.Data[0]["id"]))
	}
	return nil
}
