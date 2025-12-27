package wire

import (
	"Cornerstone/internal/api"
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/api/handler"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/kafka"
	"Cornerstone/internal/repository"
	"Cornerstone/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ApplicationContainer 封装了应用运行所需的所有顶级组件
type ApplicationContainer struct {
	Router       *gin.Engine
	DB           *gorm.DB
	KafkaManager *kafka.ConsumerManager
}

func BuildApplication(db *gorm.DB, cfg *config.Config) (*ApplicationContainer, error) {
	// 数据库 Repo 实例
	userRepo := repository.NewUserRepo(db)
	userRolesRepo := repository.NewUserRolesRepo(db)
	userFollowRepo := repository.NewUserFollowRepo(db)
	roleRepo := repository.NewRoleRepo(db)
	postRepo := repository.NewPostRepo(db)

	// Service 实例
	userService := service.NewUserService(userRepo, roleRepo)
	userRolesService := service.NewUserRolesService(userRolesRepo)
	userFollowService := service.NewUserFollowService(userFollowRepo)
	smsService := service.NewSmsService()

	// ES 实例
	userESRepo := es.NewUserRepo()
	postESRepo := es.NewPostRepo()

	handlers := &api.HandlersGroup{
		UserHandler:       handler.NewUserHandler(userService, userRolesService, smsService),
		UserFollowHandler: handler.NewUserFollowHandler(userFollowService),
	}

	router := api.SetupRouter(handlers)

	// Kafka 消费者管理
	kafkaMgr, err := kafka.NewConsumerManager(cfg, userESRepo, postESRepo, userRepo, userFollowRepo, postRepo)
	if err != nil {
		return nil, err
	}

	// 返回容器实例
	return &ApplicationContainer{
		Router:       router,
		DB:           db,
		KafkaManager: kafkaMgr,
	}, nil
}
