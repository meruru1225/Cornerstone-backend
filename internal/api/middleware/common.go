package middleware

import (
	"Cornerstone/internal/pkg/consts"
	"context"
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
)

func CommonMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		referer := c.GetHeader("Referer")
		baseURL := ""

		if referer != "" {
			if u, err := url.Parse(referer); err == nil {
				baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			}
		}

		if baseURL == "" {
			scheme := "http"
			if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
				scheme = "https"
			}
			host := c.Request.Host
			baseURL = fmt.Sprintf("%s://%s", scheme, host)
		}

		newCtx := context.WithValue(c.Request.Context(), consts.BaseURL, baseURL)
		c.Request = c.Request.WithContext(newCtx)
		c.Next()
	}
}
