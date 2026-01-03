package logger

import (
	"Cornerstone/internal/api/config"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupGin(r *gin.Engine) {
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output: LogWriter,
		Formatter: func(p gin.LogFormatterParams) string {
			var traceID string
			if p.Keys != nil {
				if id, ok := p.Keys[TraceIDKey].(string); ok {
					traceID = id
				}
			}

			if traceID == "" && p.Request != nil {
				if id, ok := p.Request.Context().Value(TraceIDKey).(string); ok {
					traceID = id
				}
			}

			return fmt.Sprintf(
				`{"time":"%s","level":"INFO","msg":"GIN_ACCESS","trace_id":"%s","log_token":"%s","target_index":"%s","method":"%s","path":"%s","status":%d,"latency":"%v"}`+"\n",
				p.TimeStamp.Format(time.RFC3339),
				traceID,
				config.Cfg.Logstash.Token,
				"logstash-cornerstone",
				p.Method,
				p.Path,
				p.StatusCode,
				p.Latency,
			)
		},
	}))

	r.Use(gin.Recovery())
}
