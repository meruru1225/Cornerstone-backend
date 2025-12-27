package middleware

import (
	"Cornerstone/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// CheckRoles 检查当前用户是否拥有至少一个指定的角色
func CheckRoles(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := c.GetStringSlice("roles")

		hasPermission := false
		for _, required := range requiredRoles {
			for _, userRole := range roles {
				if required == userRole {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			response.Fail(c, response.Forbidden, "权限不足：无权访问该资源")
			c.Abort()
			return
		}

		c.Next()
	}
}
