package redis

import (
	"context"
	"errors"
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
func TryLock(ctx context.Context, key string, value interface{}, expiration time.Duration, retryTimes int) (bool, error) {
	for i := 0; i < retryTimes || retryTimes == -1; i++ {
		success, err := Rdb.SetNX(ctx, key, value, expiration).Result()
		if err != nil {
			return false, err
		}
		if success {
			return true, nil
		}
		time.Sleep(time.Millisecond * 200)
	}
	return false, nil
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

// GetSet 获取集合
func GetSet(ctx context.Context, key string) ([]string, error) {
	value, err := Rdb.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// ZAdd 向有序集合添加一个或多个成员，或者更新已存在成员的分数
func ZAdd(ctx context.Context, key string, score float64, member string) error {
	return Rdb.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRevRange 获取有序集合中指定区间内的成员，分数从高到低排序
func ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	value, err := Rdb.ZRevRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// ZRemRangeByRank 移除有序集合中给定的排名区间的所有成员
func ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	return Rdb.ZRemRangeByRank(ctx, key, start, stop).Err()
}

func Rename(ctx context.Context, oldKey string, newKey string) error {
	return Rdb.Rename(ctx, oldKey, newKey).Err()
}

// DeleteKey 删除一个键
func DeleteKey(ctx context.Context, key string) error {
	return Rdb.Del(ctx, key).Err()
}

// GetRdbClient 获取redis客户端
func GetRdbClient() *redis.Client {
	return Rdb
}
