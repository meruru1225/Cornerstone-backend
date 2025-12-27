package kafka

import (
	log "log/slog"

	"github.com/IBM/sarama"
)

type PostConsumer struct {
}

func NewPostConsumer() *PostConsumer {
	return &PostConsumer{}
}

func (s *PostConsumer) Setup(sarama.ConsumerGroupSession) error {
	log.Info("user detail consumer setup")
	return nil
}

func (s *PostConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("user detail consumer cleanup")
	return nil
}

func (s *PostConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		// TODO 实现消费逻辑
		log.Info("consume message", "key", string(msg.Key), "value", string(msg.Value))
	}
	return nil
}
