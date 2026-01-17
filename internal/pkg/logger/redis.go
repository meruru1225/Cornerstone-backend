package logger

import (
	"context"
	"errors"
	"fmt"
	log "log/slog"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLoggerHook struct{}

func NewRedisLogger() *RedisLoggerHook {
	return &RedisLoggerHook{}
}

// DialHook 记录建立连接的事件
func (s *RedisLoggerHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		start := time.Now()
		conn, err := next(ctx, network, addr)
		elapsed := time.Since(start)

		if err != nil {
			log.ErrorContext(ctx, "Redis Dial Error",
				log.String("addr", addr),
				log.Duration("latency", elapsed),
				log.Any("err", err),
			)
		}
		return conn, err
	}
}

// ProcessHook 记录普通单条命令执行情况
func (s *RedisLoggerHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		elapsed := time.Since(start)

		cmdName := cmd.Name()

		var args string
		if cmdName == "auth" || cmdName == "hello" {
			args = "[PROTECTED]"
		} else {
			args = fmt.Sprint(cmd.Args())
		}

		fields := []any{
			log.String("command", cmdName),
			log.String("args", args),
			log.Duration("latency", elapsed),
		}

		if err != nil {
			errMsg := err.Error()
			if errors.Is(err, redis.Nil) || errMsg == "ERR no such key" {
				return err
			}
			if cmdName == "client" && strings.Contains(errMsg, "setinfo") {
				return err
			}

			log.ErrorContext(ctx, "Redis Error", append(fields, log.Any("err", err))...)
		} else {
			if elapsed > 100*time.Millisecond {
				log.WarnContext(ctx, "Redis Slow", fields...)
			}
		}

		return err
	}
}

// ProcessPipelineHook 记录管道/批量命令执行情况
func (s *RedisLoggerHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		elapsed := time.Since(start)

		if err == nil && elapsed < 100*time.Millisecond {
			return nil
		}

		if err != nil {
			log.ErrorContext(ctx, "Redis Pipeline Error",
				log.Int("cmd_count", len(cmds)),
				log.Duration("latency", elapsed),
				log.Any("err", err))
		}

		return err
	}
}
