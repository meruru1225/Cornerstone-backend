package logger

import (
	"context"
	log "log/slog"
)

// TeeHandler 将日志分发到多个 Handler
type TeeHandler struct {
	handlers []log.Handler
}

func (s *TeeHandler) Enabled(ctx context.Context, level log.Level) bool {
	return s.handlers[0].Enabled(ctx, level)
}

func (s *TeeHandler) Handle(ctx context.Context, r log.Record) error {
	for _, h := range s.handlers {
		if err := h.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *TeeHandler) WithAttrs(attrs []log.Attr) log.Handler {
	newHandlers := make([]log.Handler, len(s.handlers))
	for i, h := range s.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &TeeHandler{handlers: newHandlers}
}

func (s *TeeHandler) WithGroup(name string) log.Handler {
	newHandlers := make([]log.Handler, len(s.handlers))
	for i, h := range s.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &TeeHandler{handlers: newHandlers}
}

type RemoteFilterHandler struct {
	next log.Handler
}

func (s *RemoteFilterHandler) Enabled(ctx context.Context, level log.Level) bool {
	return s.next.Enabled(ctx, level)
}

func (s *RemoteFilterHandler) Handle(ctx context.Context, r log.Record) error {
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

	return s.next.Handle(ctx, r)
}

func (s *RemoteFilterHandler) WithAttrs(attrs []log.Attr) log.Handler {
	return &RemoteFilterHandler{next: s.next.WithAttrs(attrs)}
}

func (s *RemoteFilterHandler) WithGroup(name string) log.Handler {
	return &RemoteFilterHandler{next: s.next.WithGroup(name)}
}
