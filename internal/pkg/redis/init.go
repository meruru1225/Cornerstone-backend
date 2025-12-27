package redis

import (
	"Cornerstone/internal/api/config"
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

var Rdb *redis.Client

// InitRedis 初始化 Redis 客户端连接
func InitRedis(cfg config.RedisConfig) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,

		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	ctx := context.Background()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return err
	}

	Rdb = rdb
	return nil
}
