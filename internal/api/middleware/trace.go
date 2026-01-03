package middleware

import (
	"Cornerstone/internal/pkg/logger"
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		c.Set(logger.TraceIDKey, traceID)
		ctx := context.WithValue(c.Request.Context(), logger.TraceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Header("X-Trace-ID", traceID)
		c.Next()
	}
}
