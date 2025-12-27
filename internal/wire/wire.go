package wire

import (
	"Cornerstone/internal/api"
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/api/handler"
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
	userRepo := repository.NewUserRepo(db)
	userRolesRepo := repository.NewUserRolesRepo(db)
	userFollowRepo := repository.NewUserFollowRepo(db)
	userMetricsRepo := repository.NewUserMetricsRepository(db)
	roleRepo := repository.NewRoleRepository(db)

	userService := service.NewUserService(userRepo, roleRepo)
	userRolesService := service.NewUserRolesService(userRolesRepo)
	userFollowService := service.NewUserFollowService(userFollowRepo)
	userMetricsService := service.NewUserMetricsService(userMetricsRepo, userFollowRepo)
	smsService := service.NewSmsService()

	handlers := &api.HandlersGroup{
		UserHandler:       handler.NewUserHandler(userService, userRolesService, smsService),
		UserFollowHandler: handler.NewUserFollowHandler(userFollowService),
	}

	router := api.SetupRouter(handlers)

	kafkaMgr, err := kafka.NewConsumerManager(cfg, userMetricsService)
	if err != nil {
		return nil, err
	}

	return &ApplicationContainer{
		Router:       router,
		DB:           db,
		KafkaManager: kafkaMgr,
	}, nil
}
