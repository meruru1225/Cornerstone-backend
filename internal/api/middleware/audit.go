package middleware

import (
	"bytes"
	"io"
	log "log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	if r.body.Len() < 16384 {
		r.body.Write(b)
	}
	return r.ResponseWriter.Write(b)
}

func (r *responseBodyWriter) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		rawQuery := c.Request.URL.RawQuery
		decodedQuery, err := url.QueryUnescape(rawQuery)
		if err != nil {
			decodedQuery = rawQuery
		}

		log.InfoContext(ctx, "Recv Request",
			log.String("method", c.Request.Method),
			log.String("path", c.Request.URL.Path),
			log.String("query", decodedQuery),
			log.String("req_body", string(reqBody)),
		)

		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w
		startTime := time.Now()

		c.Next()

		log.InfoContext(ctx, "Send Response",
			log.Int("status", c.Writer.Status()),
			log.Duration("latency", time.Since(startTime)),
			log.String("res_body", w.body.String()),
		)
	}
}
