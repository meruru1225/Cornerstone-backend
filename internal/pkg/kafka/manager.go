package kafka

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/processor"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"

	"github.com/IBM/sarama"
)

// ConsumerManager 管理所有 Kafka 消费者
type ConsumerManager struct {
	usersConsumer sarama.ConsumerGroup
	usersHandler  sarama.ConsumerGroupHandler

	userDetailConsumer sarama.ConsumerGroup
	userDetailHandler  sarama.ConsumerGroupHandler

	userFollowsConsumer sarama.ConsumerGroup
	userFollowsHandler  sarama.ConsumerGroupHandler

	postConsumer sarama.ConsumerGroup
	postHandler  sarama.ConsumerGroupHandler

	commentsConsumer sarama.ConsumerGroup
	commentsHandler  sarama.ConsumerGroupHandler
}

// NewConsumerManager 构造函数
func NewConsumerManager(
	cfg *config.Config,
	contentProcessor processor.ContentLLMProcessor,
	userESRepo es.UserRepo,
	postESRepo es.PostRepo,
	userDBRepo repository.UserRepo,
	actionDBRepo repository.PostActionRepo,
	userFollowDBRepo repository.UserFollowRepo,
	postDBRepo repository.PostRepo,
) (*ConsumerManager, error) {
	saramaCfg := newSaramaConfig(cfg.Kafka)

	usersConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	usersHandler := NewUserHandler(userESRepo)

	userDetailConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserDetailConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	userDetailHandler := NewUserDetailHandler(userFollowDBRepo, userESRepo)

	userFollowsConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserFollowsConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	userFollowsHandler := NewUserFollowsConsumer()

	postsConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaPostConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	postsHandler := NewPostsHandler(userDBRepo, postDBRepo, postESRepo, contentProcessor)

	commentsConsumer, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaCommentConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	commentsHandler := NewCommentsHandler(actionDBRepo, contentProcessor)

	return &ConsumerManager{
		usersConsumer:       usersConsumer,
		usersHandler:        usersHandler,
		userDetailConsumer:  userDetailConsumer,
		userDetailHandler:   userDetailHandler,
		userFollowsConsumer: userFollowsConsumer,
		userFollowsHandler:  userFollowsHandler,
		postConsumer:        postsConsumer,
		postHandler:         postsHandler,
		commentsConsumer:    commentsConsumer,
		commentsHandler:     commentsHandler,
	}, nil
}

// Start 启动所有消费者
func (m *ConsumerManager) Start(ctx context.Context, cfg *config.Config) error {
	// 启动 User Consumer
	go func() {
		topic := cfg.KafkaUserConsumer.Topic
		log.Info("User consumer started", "topic", topic)
		for {
			if err := m.userDetailConsumer.Consume(ctx, []string{topic}, m.userDetailHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 启动 User Detail Consumer
	go func() {
		topic := cfg.KafkaUserDetailConsumer.Topic
		log.Info("User Detail consumer started", "topic", topic)
		for {
			if err := m.usersConsumer.Consume(ctx, []string{topic}, m.usersHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 启动 User Follows Consumer
	go func() {
		topic := cfg.KafkaUserFollowsConsumer.Topic
		log.Info("User Follows consumer started", "topic", topic)
		for {
			if err := m.userFollowsConsumer.Consume(ctx, []string{topic}, m.userFollowsHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// 启动 Post Consumer
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

	// 启动 Comment Consumer
	go func() {
		topic := cfg.KafkaCommentConsumer.Topic
		log.Info("Comment consumer started", "topic", topic)
		for {
			if err := m.commentsConsumer.Consume(ctx, []string{topic}, m.commentsHandler); err != nil {
				log.Error("Error from consumer", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	<-ctx.Done()
	log.Info("Kafka Manager shutting down...")

	err := m.userDetailConsumer.Close()
	if err != nil {
		log.Error("Failed to close detail consumer", "err", err)
	}
	err = m.postConsumer.Close()
	if err != nil {
		log.Error("Failed to close follows consumer", "err", err)
	}

	return nil
}
