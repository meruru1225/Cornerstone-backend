package middleware

import (
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/pkg/security"
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 负责验证 JWT 并将用户身份信息注入 Context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Fail(c, response.Unauthorized, "Token 缺失或格式错误")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		signature, err := security.ExtractSignature(tokenString)
		if err != nil {
			response.Fail(c, response.Unauthorized, "Token 缺失或格式错误")
			c.Abort()
			return
		}

		value, err := redis.GetValue(c.Request.Context(), signature)
		if err != nil {
			response.Fail(c, response.InternalServerError, "未知错误")
			c.Abort()
			return
		}
		if value != "" {
			response.Fail(c, response.Unauthorized, "Token 无效或已过期")
			c.Abort()
			return
		}

		claims, err := security.ValidateToken(tokenString)
		if err != nil {
			response.Fail(c, response.Unauthorized, "Token 无效或已过期")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)

		newCtx := context.WithValue(c.Request.Context(), "user_id", claims.UserID)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}
