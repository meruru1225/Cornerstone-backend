package kafka

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/mongo"
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

	likesConsumer sarama.ConsumerGroup
	likesHandler  sarama.ConsumerGroupHandler

	collectionsConsumer sarama.ConsumerGroup
	collectionsHandler  sarama.ConsumerGroupHandler

	viewsConsumer sarama.ConsumerGroup
	viewsHandler  sarama.ConsumerGroupHandler

	commentLikesConsumer sarama.ConsumerGroup
	commentLikesHandler  sarama.ConsumerGroupHandler
}

// NewConsumerManager 创建 Kafka 消费者管理器
func NewConsumerManager(
	cfg *config.Config,
	contentProcessor processor.ContentLLMProcessor,
	userESRepo es.UserRepo,
	postESRepo es.PostRepo,
	sysBoxRepo mongo.SysBoxRepo,
	userDBRepo repository.UserRepo,
	actionDBRepo repository.PostActionRepo,
	userFollowDBRepo repository.UserFollowRepo,
	postDBRepo repository.PostRepo,
) (*ConsumerManager, error) {
	saramaCfg := newSaramaConfig(cfg.Kafka)
	m := &ConsumerManager{}
	var err error

	// 错误回滚闭包：一旦后续初始化失败，关闭所有已打开的资源
	rollback := func() {
		m.Close()
	}

	m.usersConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserConsumer.GroupID, saramaCfg)
	if err != nil {
		return nil, err
	}
	m.usersHandler = NewUserHandler(userESRepo)

	m.userDetailConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserDetailConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.userDetailHandler = NewUserDetailHandler(userFollowDBRepo, userESRepo)

	m.userFollowsConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaUserFollowsConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.userFollowsHandler = NewUserFollowsConsumer(sysBoxRepo)

	m.postConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaPostConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.postHandler = NewPostsHandler(userDBRepo, postDBRepo, postESRepo, contentProcessor)

	m.commentsConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaCommentConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.commentsHandler = NewCommentsHandler(actionDBRepo, postDBRepo, sysBoxRepo, contentProcessor)

	m.likesConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaLikeConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.likesHandler = NewLikesHandler(postDBRepo, sysBoxRepo)

	m.collectionsConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaCollectionConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.collectionsHandler = NewCollectionsHandler(postDBRepo, sysBoxRepo)

	m.viewsConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaViewConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.viewsHandler = NewViewsHandler()

	m.commentLikesConsumer, err = sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.KafkaCommentLikeConsumer.GroupID, saramaCfg)
	if err != nil {
		rollback()
		return nil, err
	}
	m.commentLikesHandler = NewCommentLikesHandler(actionDBRepo, sysBoxRepo)

	return m, nil
}

// Start 启动所有消费者（已修复 Topic 错位问题）
func (m *ConsumerManager) Start(ctx context.Context, cfg *config.Config) error {
	// 启动各模块消费者协程
	go m.runConsumer(ctx, m.usersConsumer, cfg.KafkaUserConsumer.Topic, m.usersHandler, "User")
	go m.runConsumer(ctx, m.userDetailConsumer, cfg.KafkaUserDetailConsumer.Topic, m.userDetailHandler, "User Detail")
	go m.runConsumer(ctx, m.userFollowsConsumer, cfg.KafkaUserFollowsConsumer.Topic, m.userFollowsHandler, "User Follows")
	go m.runConsumer(ctx, m.postConsumer, cfg.KafkaPostConsumer.Topic, m.postHandler, "Post")
	go m.runConsumer(ctx, m.commentsConsumer, cfg.KafkaCommentConsumer.Topic, m.commentsHandler, "Comment")
	go m.runConsumer(ctx, m.likesConsumer, cfg.KafkaLikeConsumer.Topic, m.likesHandler, "Like")
	go m.runConsumer(ctx, m.collectionsConsumer, cfg.KafkaCollectionConsumer.Topic, m.collectionsHandler, "Collection")
	go m.runConsumer(ctx, m.viewsConsumer, cfg.KafkaViewConsumer.Topic, m.viewsHandler, "View")
	go m.runConsumer(ctx, m.commentLikesConsumer, cfg.KafkaCommentLikeConsumer.Topic, m.commentLikesHandler, "Comment Like")

	<-ctx.Done()
	log.Info("Kafka Manager shutting down...")

	m.Close() // 优雅退出
	return nil
}

// runConsumer 封装通用的消费运行逻辑
func (m *ConsumerManager) runConsumer(ctx context.Context, group sarama.ConsumerGroup, topic string, handler sarama.ConsumerGroupHandler, name string) {
	log.Info(name+" consumer started", "topic", topic)
	for {
		if err := group.Consume(ctx, []string{topic}, handler); err != nil {
			log.Error(name+" consumer loop error", "err", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// Close 统一关闭所有消费者组，释放资源
func (m *ConsumerManager) Close() {
	m.safeClose(m.usersConsumer, "Users")
	m.safeClose(m.userDetailConsumer, "User Detail")
	m.safeClose(m.userFollowsConsumer, "User Follows")
	m.safeClose(m.postConsumer, "Post")
	m.safeClose(m.commentsConsumer, "Comments")
	m.safeClose(m.likesConsumer, "Likes")
	m.safeClose(m.collectionsConsumer, "Collections")
	m.safeClose(m.viewsConsumer, "Views")
	m.safeClose(m.commentLikesConsumer, "Comment Likes")
}

// safeClose 安全关闭辅助方法
func (m *ConsumerManager) safeClose(group sarama.ConsumerGroup, name string) {
	if group != nil {
		if err := group.Close(); err != nil {
			log.Error("Failed to close consumer", "name", name, "err", err)
		} else {
			log.Info("Consumer group closed", "name", name)
		}
	}
}
