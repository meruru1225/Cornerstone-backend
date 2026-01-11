package middleware

import (
	"Cornerstone/internal/pkg/security"
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthOptionalMiddleware 可选鉴权：解析成功注入 UID，失败或缺失则 UID 为 0
func AuthOptionalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("user_id", uint64(0))
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := security.ValidateToken(token)

		if err != nil {
			c.Set("user_id", uint64(0))
		} else {
			c.Set("user_id", claims.UserID)
			newCtx := context.WithValue(c.Request.Context(), "user_id", claims.UserID)
			c.Request = c.Request.WithContext(newCtx)
		}

		c.Next()
	}
}
