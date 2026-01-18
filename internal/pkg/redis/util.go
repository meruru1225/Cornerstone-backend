package redis

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
)

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

// MGetValue 批量获取
func MGetValue(ctx context.Context, keys ...string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}
	values, err := Rdb.MGet(ctx, keys...).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	m := make(map[string]string)
	for i, val := range values {
		if val == nil {
			continue
		}
		switch v := val.(type) {
		case string:
			m[keys[i]] = v
		case []byte:
			m[keys[i]] = string(v)
		case int64:
			m[keys[i]] = strconv.FormatInt(v, 10)
		case float64:
			m[keys[i]] = strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			m[keys[i]] = strconv.FormatBool(v)
		}
	}

	return m, nil
}

// TryLock 设置键值对并设置过期时间
func TryLock(ctx context.Context, key string, value interface{}, expiration time.Duration, retryTimes int) (bool, error) {
	for i := 0; i <= retryTimes || retryTimes == -1; i++ {
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

// GetSet 获取集合
func GetSet(ctx context.Context, key string) ([]string, error) {
	value, err := Rdb.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// HGet 获取哈希表中指定字段的值
func HGet(ctx context.Context, key string, field string) (string, error) {
	value, err := Rdb.HGet(ctx, key, field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// HGetAll 获取哈希表中所有的字段和值
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	value, err := Rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// HSet 设置哈希表中的字段和值
func HSet(ctx context.Context, key string, field string, value interface{}) error {
	return Rdb.HSet(ctx, key, field, value).Err()
}

// HDel 删除哈希表 key 中的一个或多个指定域，不存在的域将被忽略
func HDel(ctx context.Context, key string, field ...string) error {
	return Rdb.HDel(ctx, key, field...).Err()
}

// SAdd 向集合添加成员
func SAdd(ctx context.Context, key string, member interface{}) error {
	return Rdb.SAdd(ctx, key, member).Err()
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

// ZRevRangeWithScores 获取有序集合中指定区间内的成员和分数，分数从高到低排序
func ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	value, err := Rdb.ZRevRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// ZRemRangeByRank 移除有序集合中给定的排名区间的所有成员
func ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	return Rdb.ZRemRangeByRank(ctx, key, start, stop).Err()
}

// Incr 自增
func Incr(ctx context.Context, key string) error {
	return Rdb.Incr(ctx, key).Err()
}

// Decr 自减
func Decr(ctx context.Context, key string) error {
	return Rdb.Decr(ctx, key).Err()
}

// GetInt64 获取 int64 类型的值
func GetInt64(ctx context.Context, key string) (int64, error) {
	val, err := Rdb.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, redis.Nil
		}
		return 0, err
	}
	return val, nil
}

func Rename(ctx context.Context, oldKey string, newKey string) error {
	return Rdb.Rename(ctx, oldKey, newKey).Err()
}

func SetWithMidnightExpiration(ctx context.Context, key string, data any) error {
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}

	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	expiration := time.Until(midnight) - time.Minute*5

	if expiration <= 0 {
		expiration = time.Minute * 1
	}

	return SetWithExpiration(ctx, key, string(bs), expiration)
}

// Publish 发布消息
func Publish(ctx context.Context, channel string, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return Rdb.Publish(ctx, channel, data).Err()
}

// Subscribe 订阅频道
func Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return Rdb.Subscribe(ctx, channels...)
}

// Exists 判断是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	result, err := Rdb.Exists(ctx, key).Result()
	return result > 0, err
}

// DeleteKey 删除一个键
func DeleteKey(ctx context.Context, key string) error {
	return Rdb.Del(ctx, key).Err()
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return Rdb.Expire(ctx, key, expiration).Err()
}

// GetRdbClient 获取redis客户端
func GetRdbClient() *redis.Client {
	return Rdb
}
