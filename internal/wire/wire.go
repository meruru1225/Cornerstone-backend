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
	"Cornerstone/internal/pkg/processor"
	"Cornerstone/internal/repository"
	"Cornerstone/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ApplicationContainer 封装了应用运行所需的所有顶级组件
type ApplicationContainer struct {
	Router       *gin.Engine
	DB           *gorm.DB
	CronMgr      *cron.Manager
	KafkaManager *kafka.ConsumerManager
}

func BuildApplication(db *gorm.DB, cfg *config.Config) (*ApplicationContainer, error) {
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

	// ES 实例
	userESRepo := es.NewUserRepo()
	postESRepo := es.NewPostRepo()

	// Agent
	toolHandler := llm.NewToolHandler(postESRepo)
	agent := llm.NewAgent(toolHandler)

	// Processor
	contentProcesser := processor.NewContentLLMProcessor()

	// Service 实例
	userService := service.NewUserService(userRepo, roleRepo, userESRepo)
	userRolesService := service.NewUserRolesService(userRolesRepo)
	userFollowService := service.NewUserFollowService(userFollowRepo)
	userMetricsService := service.NewUserMetricsService(userMetricsRepo, userFollowRepo)
	userContentMetricsService := service.NewUserContentMetricService(userContentMetricsRepo, postRepo, postActionRepo)
	smsService := service.NewSmsService()
	postService := service.NewPostService(postESRepo, postRepo, userInterestRepo)
	postActionService := service.NewPostActionService(postActionRepo, postRepo, userRepo)
	postMetricsService := service.NewPostMetricService(postMetricsRepo, postRepo)

	handlers := &api.HandlersGroup{
		AgentHandler:             handler.NewAgentHandler(agent),
		UserHandler:              handler.NewUserHandler(userService, userRolesService, smsService),
		UserFollowHandler:        handler.NewUserFollowHandler(userFollowService),
		UserMetricHandler:        handler.NewUserMetricsHandler(userMetricsService),
		PostHandler:              handler.NewPostHandler(postService),
		PostActionHandler:        handler.NewPostActionHandler(postActionService),
		PostMetricHandler:        handler.NewPostMetricHandler(postMetricsService),
		UserContentMetricHandler: handler.NewUserContentMetricHandler(userContentMetricsService),
	}

	router := api.SetupRouter(handlers)

	// Cron 任务
	userMetricsJob := job.NewUserMetricsJob(userService, userMetricsService, userFollowService)
	postMetricsJob := job.NewPostMetricsJob(postService, postMetricsService, postActionService, userContentMetricsService)
	userInterestJOb := job.NewUserInterestJob(userInterestRepo)
	cronMgr := cron.NewCronManager(userMetricsJob, postMetricsJob, userInterestJOb)

	// Kafka 消费者管理
	kafkaMgr, err := kafka.NewConsumerManager(cfg, contentProcesser, userESRepo, postESRepo, userRepo, postActionRepo, userFollowRepo, postRepo)
	if err != nil {
		return nil, err
	}

	// 返回容器实例
	return &ApplicationContainer{
		Router:       router,
		DB:           db,
		CronMgr:      cronMgr,
		KafkaManager: kafkaMgr,
	}, nil
}
