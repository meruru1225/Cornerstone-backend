package middleware

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/pkg/security"
	"context"
	"strconv"
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

		// 验证 Token
		claims, err := security.ValidateToken(tokenString)
		if err != nil {
			response.Fail(c, response.Unauthorized, "Token 无效或已过期")
			c.Abort()
			return
		}

		// 检查 Token 是否已注销
		signature, err := security.ExtractSignature(tokenString)
		if err == nil {
			value, _ := redis.GetValue(c.Request.Context(), signature)
			if value != "" {
				response.Fail(c, response.Unauthorized, "Token 已注销")
				c.Abort()
				return
			}
		}

		// 检查用户权限是否已更新
		key := consts.UserAuthVersionKey + strconv.FormatUint(claims.UserID, 10)
		verVal, _ := redis.GetValue(c.Request.Context(), key)
		if verVal != "" {
			lastInvalidateTime, err := strconv.ParseInt(verVal, 10, 64)
			if err == nil && lastInvalidateTime >= claims.IssuedAt.Time.Unix() {
				response.Fail(c, response.Unauthorized, "权限已更新，请重新登录")
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)

		newCtx := context.WithValue(c.Request.Context(), "user_id", claims.UserID)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}
