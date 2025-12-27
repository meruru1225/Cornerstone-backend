package kafka

import (
	"Cornerstone/internal/pkg/es"
	"context"
	"errors"
	log "log/slog"

	"github.com/IBM/sarama"
)

type UserHandler struct {
	userESRepo es.UserRepo
}

func NewUserHandler(userESRepo es.UserRepo) *UserDetailHandler {
	return &UserDetailHandler{
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
	canalMsg, err := ToCanalMessage(msg, "user_detail")
	if err != nil {
		return err
	}
	if len(canalMsg.Data) == 0 {
		return errors.New("canal message data is empty")
	}

	if canalMsg.Data[0]["is_deleted"] == "1" {
		return s.userESRepo.DeleteUser(ctx, StrToUint64(canalMsg.Data[0]["id"]))
	}
	return nil
}
