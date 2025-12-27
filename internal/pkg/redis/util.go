package redis

import (
	"context"
	"errors"
	log "log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// SetValue 设置键值对
func SetValue(ctx context.Context, key string, value interface{}) error {
	return Rdb.Set(ctx, key, value, 0).Err()
}

// SetWithExpiration 设置键值对并设置过期时间
func SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return Rdb.Set(ctx, key, value, expiration).Err()
}

// GetValue 获取字符串类型的值
func GetValue(ctx context.Context, key string) (string, error) {
	value, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// TryLock 设置键值对并设置过期时间
func TryLock(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	isSet := Rdb.SetNX(ctx, key, value, expiration)
	if isSet.Err() != nil {
		log.Error("set key failed", "key", key, "err", isSet.Err())
		return false, isSet.Err()
	}
	return isSet.Val(), nil
}

// UnLock 释放锁
func UnLock(ctx context.Context, key string, value interface{}) {
	Rdb.Eval(ctx, "if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end", []string{key}, value)
}

// SetList 设置列表
func SetList(ctx context.Context, key string, value []string) error {
	return Rdb.RPush(ctx, key, value).Err()
}

// SetListWithExpiration 设置列表并设置过期时间
func SetListWithExpiration(ctx context.Context, key string, value []string, expiration time.Duration) error {
	pipe := Rdb.TxPipeline()
	pipe.RPush(ctx, key, value)
	pipe.Expire(ctx, key, expiration)
	_, err := pipe.Exec(ctx)
	return err
}

// GetList 获取列表
func GetList(ctx context.Context, key string) ([]string, error) {
	value, err := Rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return value, nil
}

// DeleteKey 删除一个键
func DeleteKey(ctx context.Context, key string) error {
	return Rdb.Del(ctx, key).Err()
}

// GetRdbClient 获取redis客户端
func GetRdbClient() *redis.Client {
	return Rdb
}
