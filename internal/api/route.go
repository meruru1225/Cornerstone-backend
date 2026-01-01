package api

import (
	"Cornerstone/internal/api/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(group *HandlersGroup) *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies([]string{"localhost"})

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
			userGroup.POST("/cancelUser", group.UserHandler.CancelUser)
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
	}

	return r
}
