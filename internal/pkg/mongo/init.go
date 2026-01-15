package mongo

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/logger"
	"context"
	log "log/slog"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitMongo 建立连接并返回 Database 引用，同时初始化 Schema
func InitMongo(cfg config.MongoConfig) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 建立连接
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI(cfg.URL).
		SetMonitor(logger.NewMongoMonitor()),
	)
	if err != nil {
		return nil, err
	}

	// 检查连通性
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(cfg.Database)

	log.Info("MongoDB initialized successfully", "db", cfg.Database)
	return db, nil
}
