package wire

import (
	"Cornerstone/internal/api"
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/api/handler"
	"Cornerstone/internal/job"
	"Cornerstone/internal/pkg/cron"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/kafka"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/mongo"
	"Cornerstone/internal/pkg/processor"
	"Cornerstone/internal/repository"
	"Cornerstone/internal/service"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	mongoDrive "go.mongodb.org/mongo-driver/mongo"

	"gorm.io/gorm"
)

// ApplicationContainer 封装了应用运行所需的所有顶级组件
type ApplicationContainer struct {
	Router       *gin.Engine
	CronMgr      *cron.Manager
	KafkaManager *kafka.ConsumerManager
}

func BuildApplication(
	db *gorm.DB,
	elasticClient *elasticsearch.TypedClient,
	mongoConn *mongoDrive.Database,
	cfg *config.Config,
) (*ApplicationContainer, error) {
	// 数据库 Repo 实例
	userRepo := repository.NewUserRepo(db)
	userRolesRepo := repository.NewUserRolesRepo(db)
	userFollowRepo := repository.NewUserFollowRepo(db)
	userMetricsRepo := repository.NewUserMetricsRepository(db)
	userContentMetricsRepo := repository.NewUserContentMetricRepository(db)
	roleRepo := repository.NewRoleRepo(db)
	postRepo := repository.NewPostRepo(db)
	postActionRepo := repository.NewPostActionRepo(db)
	postMetricsRepo := repository.NewPostMetricRepository(db)
	userInterestRepo := repository.NewUserInterestRepository(db)
	conversationRepo := repository.NewConversationRepo(db)

	// Mongo 实例
	messageMongoRepo := mongo.NewMessageRepo(mongoConn)
	sysBoxRepo := mongo.NewSysBoxRepo(mongoConn)
	agentMessageRepo := mongo.NewAgentMessageRepo(mongoConn)

	// ES 实例
	userESRepo := es.NewUserRepo(elasticClient)
	postESRepo := es.NewPostRepo(elasticClient)

	// Agent
	toolHandler := llm.NewToolHandler(postESRepo)
	agent := llm.NewAgent(toolHandler, agentMessageRepo)

	// Processor
	contentProcesser := processor.NewContentLLMProcessor()

	// Service 实例
	userService := service.NewUserService(userRepo, roleRepo, userRolesRepo, userESRepo)
	userRolesService := service.NewUserRolesService(userRolesRepo)
	userFollowService := service.NewUserFollowService(userRepo, userFollowRepo)
	userMetricsService := service.NewUserMetricsService(userMetricsRepo, userFollowRepo)
	userContentMetricsService := service.NewUserContentMetricService(userContentMetricsRepo, postRepo, postActionRepo)
	smsService := service.NewSmsService()
	postService := service.NewPostService(postESRepo, postRepo, userInterestRepo)
	postActionService := service.NewPostActionService(postActionRepo, postRepo, userRepo)
	postMetricsService := service.NewPostMetricService(postMetricsRepo, postRepo)
	IMService := service.NewIMService(userRepo, conversationRepo, messageMongoRepo)
	sysBoxService := service.NewSysBoxService(sysBoxRepo, userRepo)

	handlers := &api.HandlersGroup{
		AgentHandler:             handler.NewAgentHandler(agent),
		UserHandler:              handler.NewUserHandler(userService, userRolesService, smsService),
		UserFollowHandler:        handler.NewUserFollowHandler(userFollowService),
		UserMetricHandler:        handler.NewUserMetricsHandler(userMetricsService),
		PostHandler:              handler.NewPostHandler(postService),
		PostActionHandler:        handler.NewPostActionHandler(postService, postActionService),
		PostMetricHandler:        handler.NewPostMetricHandler(postMetricsService),
		UserContentMetricHandler: handler.NewUserContentMetricHandler(userContentMetricsService),
		IMHandler:                handler.NewIMHandler(IMService),
		WSHandler:                handler.NewWsHandler(IMService),
		SysBoxHandler:            handler.NewSysBoxHandler(sysBoxService),
		MediaHandler:             handler.NewMediaHandler(),
	}

	router := api.SetupRouter(handlers)

	// Cron 任务
	userMetricsJob := job.NewUserMetricsJob(userService, userMetricsService, userFollowService)
	postMetricsJob := job.NewPostMetricsJob(postService, postMetricsService, postActionService, userContentMetricsService)
	userInterestJOb := job.NewUserInterestJob(userInterestRepo)
	postCommentJob := job.NewPostCommentJob(postActionService)
	mediaCleanJob := job.NewMediaCleanupJob()
	cronMgr := cron.NewCronManager(userMetricsJob, postMetricsJob, userInterestJOb, postCommentJob, mediaCleanJob)

	// Kafka 消费者管理
	kafkaMgr, err := kafka.NewConsumerManager(cfg, contentProcesser, userESRepo, postESRepo, sysBoxRepo,
		userRepo, postActionRepo, userFollowRepo, postRepo)
	if err != nil {
		return nil, err
	}

	// 返回容器实例
	return &ApplicationContainer{
		Router:       router,
		CronMgr:      cronMgr,
		KafkaManager: kafkaMgr,
	}, nil
}
