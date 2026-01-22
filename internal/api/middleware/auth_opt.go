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
		var userID uint64
		var token string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if token == "" {
			if cookieToken, err := c.Cookie("auth_token"); err == nil {
				token = cookieToken
			}
		}

		if token != "" {
			if claims, err := security.ValidateToken(token); err == nil {
				userID = claims.UserID
			}
		}

		c.Set("user_id", userID)

		newCtx := context.WithValue(c.Request.Context(), "user_id", userID)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}
