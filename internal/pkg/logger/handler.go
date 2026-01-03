package logger

import (
	"context"
	log "log/slog"
)

// TeeHandler 将日志分发到多个 Handler
type TeeHandler struct {
	handlers []log.Handler
}

func (t *TeeHandler) Enabled(ctx context.Context, level log.Level) bool {
	return t.handlers[0].Enabled(ctx, level)
}

func (t *TeeHandler) Handle(ctx context.Context, r log.Record) error {
	for _, h := range t.handlers {
		if err := h.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (t *TeeHandler) WithAttrs(attrs []log.Attr) log.Handler {
	newHandlers := make([]log.Handler, len(t.handlers))
	for i, h := range t.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &TeeHandler{handlers: newHandlers}
}

func (t *TeeHandler) WithGroup(name string) log.Handler {
	newHandlers := make([]log.Handler, len(t.handlers))
	for i, h := range t.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &TeeHandler{handlers: newHandlers}
}

type RemoteFilterHandler struct {
	next log.Handler
}

func (h *RemoteFilterHandler) Enabled(ctx context.Context, level log.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *RemoteFilterHandler) Handle(ctx context.Context, r log.Record) error {
	// 检查日志记录中是否存在 trace_id 属性
	hasTraceID := false
	r.Attrs(func(a log.Attr) bool {
		if a.Key == TraceIDKey && a.Value.String() != "" {
			hasTraceID = true
			return false
		}
		return true
	})

	// 如果没有 trace_id，则跳过远程上报
	if !hasTraceID {
		return nil
	}

	return h.next.Handle(ctx, r)
}

func (h *RemoteFilterHandler) WithAttrs(attrs []log.Attr) log.Handler {
	return &RemoteFilterHandler{next: h.next.WithAttrs(attrs)}
}

func (h *RemoteFilterHandler) WithGroup(name string) log.Handler {
	return &RemoteFilterHandler{next: h.next.WithGroup(name)}
}
