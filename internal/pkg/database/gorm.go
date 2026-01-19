package database

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/logger"
	"fmt"
	log "log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewGormDB 初始化并返回 *gorm.DB 实例，处理连接池配置
func NewGormDB(cfg *config.DBConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	dialector = mysql.Open(cfg.DSN)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:      logger.NewGormLogger(),
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Minute)

	if err = sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database connection check failed: %w", err)
	}

	log.Info("Database connection established successfully.")
	return db, nil
}
