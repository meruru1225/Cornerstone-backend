package api

import (
	"Cornerstone/internal/api/middleware"
	"Cornerstone/internal/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(group *HandlersGroup) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies([]string{"localhost"})

	// TraceId & Logger
	r.Use(middleware.TraceMiddleware())
	r.Use(middleware.AuditMiddleware())
	logger.SetupGin(r)

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"Code":    200,
				"Message": "pong",
				"Data":    nil,
			})
		})

		agentGroup := apiGroup.Group("/agent")
		{
			agentGroup.GET("/search", group.AgentHandler.Search)
		}

		userGroup := apiGroup.Group("/user")
		{
			// 无需登录即可访问的接口
			userGroup.POST("/login", group.UserHandler.Login)
			userGroup.POST("/loginByPhone", group.UserHandler.LoginByPhone)
			userGroup.POST("/register", group.UserHandler.Register)
			userGroup.GET("/sendSmsCode", group.UserHandler.SendSmsCode)
			userGroup.PUT("/forgetPassword", group.UserHandler.ForgetPassword)
			userGroup.GET("/homeInfo", group.UserHandler.GetHomeInfo)
			userGroup.GET("/simpleInfo", group.UserHandler.GetUserSimpleInfoById)
			userGroup.GET("/simpleInfos", group.UserHandler.GetUserSimpleInfoByIds)

			authGroup := userGroup.Group("")
			authGroup.Use(middleware.AuthMiddleware())
			{
				authGroup.POST("/logout", group.UserHandler.Logout)
				authGroup.GET("/info", group.UserHandler.GetUserInfo)
				authGroup.PUT("/info", group.UserHandler.UpdateUserInfo)
				authGroup.PUT("/password", group.UserHandler.ChangePassword)
				authGroup.PUT("/username", group.UserHandler.ChangeUsername)
				authGroup.PUT("/phone", group.UserHandler.ChangePhone)
				authGroup.POST("/avatar", group.UserHandler.UploadAvatar)
				userGroup.POST("/cancelUser", group.UserHandler.CancelUser)
			}

			// 需要登录 & 拥有 admin 角色
			adminGroup := authGroup.Group("")
			adminGroup.Use(middleware.CheckRoles("admin"))
			{
				adminGroup.POST("/ban", group.UserHandler.BanUser)
				adminGroup.POST("/unban", group.UserHandler.UnbanUser)
				adminGroup.POST("/searchUser", group.UserHandler.SearchUser)
				adminGroup.POST("/userRole", group.UserHandler.AddUserRole)
				adminGroup.DELETE("/userRole", group.UserHandler.DeleteUserRole)
			}
		}

		userFollowGroup := apiGroup.Group("/user-relation")
		{
			userFollowGroup.Use(middleware.AuthMiddleware())
			{
				userFollowGroup.GET("/followers", group.UserFollowHandler.GetUserFollowers)
				userFollowGroup.GET("/followersCount", group.UserFollowHandler.GetUserFollowersCount)
				userFollowGroup.GET("/followings", group.UserFollowHandler.GetUserFollowings)
				userFollowGroup.GET("/followingsCount", group.UserFollowHandler.GetUserFollowingCount)
				userFollowGroup.GET("/isFollow", group.UserFollowHandler.GetSomeoneIsFollowing)
				userFollowGroup.POST("/followUser", group.UserFollowHandler.Follow)
				userFollowGroup.DELETE("/followUser", group.UserFollowHandler.Unfollow)
			}
		}

		userMetricsGroup := apiGroup.Group("/user-metrics")
		{
			userMetricsGroup.Use(middleware.AuthMiddleware())
			{
				userMetricsGroup.GET("/7d", group.UserMetricHandler.GetMetrics7Days)
				userMetricsGroup.GET("/30d", group.UserMetricHandler.GetMetrics30Days)
			}
		}

		postGroup := apiGroup.Group("/post")
		{
			authOptGroup := postGroup.Group("")
			authOptGroup.Use(middleware.AuthOptionalMiddleware())
			{
				authOptGroup.GET("/recommend", group.PostHandler.RecommendPost)
				authOptGroup.GET("/search", group.PostHandler.SearchPost)
				authOptGroup.GET("/detail/:post_id", group.PostHandler.GetPost)
			}

			authGroup := postGroup.Group("")
			authGroup.Use(middleware.AuthMiddleware())
			{
				authGroup.POST("", group.PostHandler.CreatePost)
				authGroup.PUT("/:post_id", group.PostHandler.UpdatePostContent)
				authGroup.DELETE("/:post_id", group.PostHandler.DeletePost)
				authGroup.GET("/self", group.PostHandler.GetPostSelf)
			}
		}
	}

	return r
}
