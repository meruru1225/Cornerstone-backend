package logger

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

type SlogGormLogger struct {
	LogLevel logger.LogLevel
}

func NewGormLogger() *SlogGormLogger {
	return &SlogGormLogger{LogLevel: logger.Info}
}

func (l *SlogGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l
}

func (l *SlogGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		slog.InfoContext(ctx, msg, "data", data)
	}
}

func (l *SlogGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		slog.WarnContext(ctx, msg, "data", data)
	}
}

func (l *SlogGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		slog.ErrorContext(ctx, msg, "data", data)
	}
}

func (l *SlogGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	operation := "Query"
	if len(sql) > 0 {
		for i, char := range sql {
			if char == ' ' {
				operation = sql[:i]
				break
			}
		}
	}
	msg := "MySQL " + operation

	fields := []any{
		slog.String("sql", sql),
		slog.Duration("latency", elapsed),
		slog.Int64("rows", rows),
	}

	if err != nil && !errors.Is(err, logger.ErrRecordNotFound) {
		slog.ErrorContext(ctx, msg+" Error", append(fields, slog.Any("err", err))...)
	} else if elapsed > 200*time.Millisecond {
		slog.WarnContext(ctx, msg+" Slow", fields...)
	} else {
		slog.InfoContext(ctx, msg, fields...)
	}
}
