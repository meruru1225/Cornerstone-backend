package logger

import (
	"context"
	log "log/slog"
	"time"

	"gorm.io/gorm/logger"
)

type GormSlogAdapter struct{}

func (g *GormSlogAdapter) LogMode(logger.LogLevel) logger.Interface {
	return g
}

func (g *GormSlogAdapter) Info(context.Context, string, ...interface{}) {

}

func (g *GormSlogAdapter) Warn(context.Context, string, ...interface{}) {

}

func (g *GormSlogAdapter) Error(context.Context, string, ...interface{}) {

}

// Trace 捕获 SQL 执行明细
func (g *GormSlogAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), _ error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	log.InfoContext(ctx, "SQL_TRACE",
		log.Duration("latency", elapsed),
		log.String("sql", sql),
		log.Int64("rows", rows),
	)
}
