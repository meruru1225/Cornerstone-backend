package kafka

import (
	"Cornerstone/internal/api/config"
	"context"
	log "log/slog"

	"github.com/IBM/sarama"
)

// ConsumerManager 管理所有 Kafka 消费者
type ConsumerManager struct {
	userConsumer sarama.ConsumerGroup
	userHandler  sarama.ConsumerGroupHandler

	postConsumer sarama.ConsumerGroup
	postHandler  sarama.ConsumerGroupHandler
}

// NewConsumerManager 构造函数
func NewConsumerManager(cfg *config.Config) (*ConsumerManager, error) {
	saramaCfg := newSaramaConfig(cfg.Kafka)

	userConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	userHandler := NewUserConsumer()

	postConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaPostConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	postHandler := NewPostConsumer()

	return &ConsumerManager{
		userConsumer: userConsumer,
		userHandler:  userHandler,
		postConsumer: postConsumer,
		postHandler:  postHandler,
	}, nil
}

// Start 启动所有消费者
func (m *ConsumerManager) Start(ctx context.Context, cfg *config.Config) error {
	go func() {
		topic := cfg.KafkaUserConsumer.Topic
		log.Info("User consumer started", "topic", topic)
		for {
			if err := m.userConsumer.Consume(ctx, []string{topic}, m.userHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 启动 Follows Consumer
	go func() {
		topic := cfg.KafkaPostConsumer.Topic
		log.Info("Post consumer started", "topic", topic)
		for {
			if err := m.postConsumer.Consume(ctx, []string{topic}, m.postHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	<-ctx.Done()
	log.Info("Kafka Manager shutting down...")

	err := m.userConsumer.Close()
	if err != nil {
		log.Error("Failed to close detail consumer", "err", err)
	}
	err = m.postConsumer.Close()
	if err != nil {
		log.Error("Failed to close follows consumer", "err", err)
	}

	return nil
}
