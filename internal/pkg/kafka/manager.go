package kafka

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/service"
	"context"
	log "log/slog"

	"github.com/IBM/sarama"
)

// ConsumerManager 管理所有 Kafka 消费者
type ConsumerManager struct {
	detailConsumer sarama.ConsumerGroup
	detailHandler  sarama.ConsumerGroupHandler

	followsConsumer sarama.ConsumerGroup
	followsHandler  sarama.ConsumerGroupHandler
}

// NewConsumerManager 构造函数
func NewConsumerManager(
	cfg *config.Config,
	userMetricsService service.UserMetricsService,
) (*ConsumerManager, error) {
	saramaCfg := newSaramaConfig(cfg.Kafka)

	detailConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserDetailConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	detailHandler := NewUserDetailConsumer()

	followsConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserFollowsConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	followsHandler := NewUserFollowsConsumer(userMetricsService)

	return &ConsumerManager{
		detailConsumer:  detailConsumer,
		detailHandler:   detailHandler,
		followsConsumer: followsConsumer,
		followsHandler:  followsHandler,
	}, nil
}

// Start 启动所有消费者
func (m *ConsumerManager) Start(ctx context.Context, cfg *config.Config) error {
	go func() {
		topic := cfg.KafkaUserDetailConsumer.Topic
		log.Info("UserDetail consumer started", "topic", topic)
		for {
			if err := m.detailConsumer.Consume(ctx, []string{topic}, m.detailHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 启动 Follows Consumer
	go func() {
		topic := cfg.KafkaUserFollowsConsumer.Topic
		log.Info("UserFollows consumer started", "topic", topic)
		for {
			if err := m.followsConsumer.Consume(ctx, []string{topic}, m.followsHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	<-ctx.Done()
	log.Info("Kafka Manager shutting down...")

	err := m.detailConsumer.Close()
	if err != nil {
		log.Error("Failed to close detail consumer", "err", err)
	}
	err = m.followsConsumer.Close()
	if err != nil {
		log.Error("Failed to close follows consumer", "err", err)
	}

	return nil
}
