package logger

import (
	"context"
	"fmt"
	log "log/slog"
	"time"

	"go.mongodb.org/mongo-driver/event"
)

func NewMongoMonitor() *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(ctx context.Context, evt *event.CommandStartedEvent) {
			cmdStr := evt.Command.String()
			if len(cmdStr) > 1000 {
				cmdStr = cmdStr[:1000] + "...[truncated]"
			}

			log.InfoContext(ctx, "MongoDB Started",
				log.String("command", evt.CommandName),
				log.String("database", evt.DatabaseName),
				log.String("request_id", fmt.Sprintf("%d", evt.RequestID)),
				log.String("cmd_detail", cmdStr),
			)
		},
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			fields := []any{
				log.String("command", evt.CommandName),
				log.Duration("latency", evt.Duration),
				log.String("request_id", fmt.Sprintf("%d", evt.RequestID)),
			}

			if evt.Duration > 200*time.Millisecond {
				log.WarnContext(ctx, "MongoDB Slow", fields...)
			} else {
				log.InfoContext(ctx, "MongoDB Success", fields...)
			}
		},
		Failed: func(ctx context.Context, evt *event.CommandFailedEvent) {
			log.ErrorContext(ctx, "MongoDB Error",
				log.String("command", evt.CommandName),
				log.Duration("latency", evt.Duration),
				log.String("request_id", fmt.Sprintf("%d", evt.RequestID)),
				log.Any("err", evt.Failure),
			)
		},
	}
}
