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

	// TraceId & Logger & CORS
	r.Use(middleware.TraceMiddleware())
	r.Use(middleware.AuditMiddleware())
	r.Use(middleware.CORSMiddleware())
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

			authGroup := agentGroup.Group("")
			authGroup.Use(middleware.AuthOptionalMiddleware())
			{
				authGroup.POST("/converse", group.AgentHandler.Converse)
			}
		}

		userGroup := apiGroup.Group("/user")
		{
			// 无需登录即可访问的接口
			userGroup.POST("/login", group.UserHandler.Login)
			userGroup.POST("/login/phone", group.UserHandler.LoginByPhone)
			userGroup.POST("/register", group.UserHandler.Register)
			userGroup.POST("/sms/send", group.UserHandler.SendSmsCode)
			userGroup.PUT("/password/forget", group.UserHandler.ForgetPassword)
			userGroup.GET("/:user_id/home", group.UserHandler.GetHomeInfo)
			userGroup.GET("/:user_id/simple", group.UserHandler.GetUserSimpleInfoById)
			userGroup.GET("/batch/simple", group.UserHandler.GetUserSimpleInfoByIds)
			userGroup.GET("/search", group.UserHandler.SearchUser)

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
				authGroup.POST("/cancel", group.UserHandler.CancelUser)
			}

			// 需要登录 & 拥有 admin 角色
			adminGroup := authGroup.Group("")
			adminGroup.Use(middleware.CheckRoles("ADMIN"))
			{
				adminGroup.POST("/ban", group.UserHandler.BanUser)
				adminGroup.POST("/unban", group.UserHandler.UnbanUser)
				adminGroup.GET("/condition", group.UserHandler.GetUserByCondition)
				adminGroup.GET("/roles", group.UserHandler.GetAllRoles)
				adminGroup.POST("/role", group.UserHandler.AddUserRole)
				adminGroup.DELETE("/role", group.UserHandler.DeleteUserRole)
			}
		}

		userFollowGroup := apiGroup.Group("/user-relation")
		{
			userFollowGroup.Use(middleware.AuthMiddleware())
			{
				userFollowGroup.GET("/followers", group.UserFollowHandler.GetUserFollowers)
				userFollowGroup.GET("/followers/count", group.UserFollowHandler.GetUserFollowersCount)
				userFollowGroup.GET("/followings", group.UserFollowHandler.GetUserFollowings)
				userFollowGroup.GET("/followings/count", group.UserFollowHandler.GetUserFollowingCount)
				userFollowGroup.GET("/isfollow/:following_id", group.UserFollowHandler.GetSomeoneIsFollowing)
				userFollowGroup.POST("/follow/:following_id", group.UserFollowHandler.Follow)
				userFollowGroup.DELETE("/follow/:following_id", group.UserFollowHandler.Unfollow)
			}
		}

		metricsGroup := apiGroup.Group("/metrics")
		{
			metricsGroup.Use(middleware.AuthMiddleware())
			{
				metricsGroup.GET("/user/7d", group.UserMetricHandler.GetMetrics7Days)
				metricsGroup.GET("/user/30d", group.UserMetricHandler.GetMetrics30Days)
				metricsGroup.GET("/user-content/7d", group.UserContentMetricHandler.GetMetrics7Days)
				metricsGroup.GET("/user-content/30d", group.UserContentMetricHandler.GetMetrics30Days)
				metricsGroup.GET("/post/7d/:post_id", group.PostMetricHandler.GetMetrics7Days)
				metricsGroup.GET("/post/30d/:post_id", group.PostMetricHandler.GetMetrics30Days)
			}
		}

		postGroup := apiGroup.Group("/posts")
		{
			authOptGroup := postGroup.Group("")
			authOptGroup.Use(middleware.AuthOptionalMiddleware())
			{
				authOptGroup.GET("/recommend", group.PostHandler.RecommendPost)
				authOptGroup.GET("/search", group.PostHandler.SearchPost)
				authOptGroup.GET("/detail/:post_id", group.PostHandler.GetPost)
				authOptGroup.GET("/list/:user_id", group.PostHandler.GetPostByUserId)
			}

			authGroup := postGroup.Group("")
			authGroup.Use(middleware.AuthMiddleware())
			{
				authGroup.POST("", group.PostHandler.CreatePost)
				authGroup.PUT("/:post_id", group.PostHandler.UpdatePostContent)
				authGroup.DELETE("/:post_id", group.PostHandler.DeletePost)
				authGroup.GET("/self", group.PostHandler.GetPostSelf)
			}

			auditGroup := authGroup.Group("/audit")
			auditGroup.Use(middleware.AuthMiddleware(), middleware.CheckRoles("AUDIT", "ADMIN"))
			{
				auditGroup.GET("/list", group.PostHandler.GetWarningPosts)
				auditGroup.PUT("/:post_id/status", group.PostHandler.UpdatePostStatus)
			}
		}

		postActionGroup := apiGroup.Group("/post/action")
		{
			postActionGroup.GET("/comments/:post_id", group.PostActionHandler.GetComments)
			postActionGroup.GET("/sub-comments/:root_id", group.PostActionHandler.GetSubComments)
			postActionGroup.POST("/batch/likes", group.PostActionHandler.GetBatchLikes)

			authActionGroup := postActionGroup.Group("")
			authActionGroup.Use(middleware.AuthMiddleware())
			{
				authActionGroup.POST("/likes/:post_id", group.PostActionHandler.LikePost)
				authActionGroup.POST("/collects/:post_id", group.PostActionHandler.CollectPost)
				authActionGroup.GET("/state/:post_id", group.PostActionHandler.GetPostActionState)

				authActionGroup.POST("/comments", group.PostActionHandler.CreateComment)
				authActionGroup.DELETE("/comments/:comment_id", group.PostActionHandler.DeleteComment)
				authActionGroup.POST("/comments/:comment_id/like", group.PostActionHandler.LikeComment)

				authActionGroup.GET("/liked", group.PostActionHandler.GetUserLikes)
				authActionGroup.GET("/collections", group.PostActionHandler.GetUserCollections)

				authActionGroup.POST("/reports/:post_id", group.PostActionHandler.ReportPost)
			}
		}

		imGroup := apiGroup.Group("/im")
		{
			imGroup.GET("", group.WSHandler.Connect)
			authGroup := imGroup.Group("")
			authGroup.Use(middleware.AuthMiddleware())
			{
				authGroup.POST("/send", group.IMHandler.SendMessage)
				authGroup.GET("/history", group.IMHandler.GetChatHistory)
				authGroup.GET("/sync", group.IMHandler.GetNewMessages)
				authGroup.GET("/list", group.IMHandler.GetConversationList)
				authGroup.POST("/read", group.IMHandler.MarkAsRead)
			}
		}

		sysbox := apiGroup.Group("/sysbox")
		sysbox.Use(middleware.AuthMiddleware())
		{
			sysbox.GET("/list", group.SysBoxHandler.GetNotificationList)
			sysbox.GET("/unread", group.SysBoxHandler.GetUnreadCount)
			sysbox.POST("/read", group.SysBoxHandler.MarkRead)
			sysbox.POST("/read/all", group.SysBoxHandler.MarkAllRead)
		}

		mediaGroup := apiGroup.Group("/media")
		{
			mediaGroup.Use(middleware.AuthMiddleware())
			mediaGroup.POST("/upload", group.MediaHandler.Upload)
		}
	}

	return r
}
