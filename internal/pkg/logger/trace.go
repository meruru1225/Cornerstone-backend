package logger

import (
	"context"
	log "log/slog"
)

// TraceIDKey 定义 Context 中的 Key
const TraceIDKey = "trace_id"

// ContextHandler 包装器，用于从 ctx 中提取 trace_id
type ContextHandler struct {
	log.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r log.Record) error {
	if ctx != nil {
		if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
			r.AddAttrs(log.String("trace_id", traceID))
		}
	}
	return h.Handler.Handle(ctx, r)
}
