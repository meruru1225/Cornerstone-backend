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
				userFollowGroup.GET("/followersCount", group.UserFollowHandler.GetUserFollowersCount)
				userFollowGroup.GET("/followings", group.UserFollowHandler.GetUserFollowings)
				userFollowGroup.GET("/followingsCount", group.UserFollowHandler.GetUserFollowingCount)
				userFollowGroup.GET("/isFollow/:following_id", group.UserFollowHandler.GetSomeoneIsFollowing)
				userFollowGroup.POST("/followUser", group.UserFollowHandler.Follow)
				userFollowGroup.DELETE("/followUser", group.UserFollowHandler.Unfollow)
			}
		}

		metricsGroup := apiGroup.Group("/metrics")
		{
			metricsGroup.Use(middleware.AuthMiddleware())
			{
				metricsGroup.GET("/user-7d", group.UserMetricHandler.GetMetrics7Days)
				metricsGroup.GET("/user-30d", group.UserMetricHandler.GetMetrics30Days)
				metricsGroup.GET("/user-content-7d", group.UserContentMetricHandler.GetMetrics7Days)
				metricsGroup.GET("/user-content-30d", group.UserContentMetricHandler.GetMetrics30Days)
				metricsGroup.GET("/post-7d", group.PostMetricHandler.GetMetrics7Days)
				metricsGroup.GET("/post-30d", group.PostMetricHandler.GetMetrics30Days)
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
				auditGroup.POST("/status", group.PostHandler.UpdatePostStatus)
			}
		}

		postActionGroup := apiGroup.Group("/post-action")
		{
			postActionGroup.GET("/comments", group.PostActionHandler.GetComments)
			postActionGroup.GET("/sub-comments", group.PostActionHandler.GetSubComments)
			postActionGroup.POST("/batch-likes", group.PostActionHandler.GetBatchLikes)

			authActionGroup := postActionGroup.Group("")
			authActionGroup.Use(middleware.AuthMiddleware())
			{
				authActionGroup.POST("/like", group.PostActionHandler.LikePost)
				authActionGroup.POST("/collect", group.PostActionHandler.CollectPost)
				authActionGroup.GET("/state", group.PostActionHandler.GetPostActionState)

				authActionGroup.POST("/comment", group.PostActionHandler.CreateComment)
				authActionGroup.DELETE("/comment/:comment_id", group.PostActionHandler.DeleteComment)
				authActionGroup.POST("/comment/like", group.PostActionHandler.LikeComment)

				authActionGroup.GET("/my/likes", group.PostActionHandler.GetUserLikes)
				authActionGroup.GET("/my/collections", group.PostActionHandler.GetUserCollections)

				authActionGroup.POST("/report", group.PostActionHandler.ReportPost)
			}
		}

		imGroup := apiGroup.Group("/im")
		{
			imGroup.GET("", group.WSHandler.Connect)
			authGroup := imGroup.Group("")
			authGroup.Use(middleware.AuthMiddleware())
			{
				authGroup.POST("/message/send", group.IMHandler.SendMessage)
				authGroup.GET("/message/history", group.IMHandler.GetChatHistory)
				authGroup.GET("/conversations", group.IMHandler.GetConversationList)
				authGroup.POST("/message/read", group.IMHandler.MarkAsRead)
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
			mediaGroup.POST("/upload", group.MediaHandler.Upload)
		}
	}

	return r
}
