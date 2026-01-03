package kafka

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/logger"
	"context"
	"errors"
	log "log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

type LogicFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

// pullMessageBatch 拉取一批消息并执行业务逻辑
func pullMessageBatch(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim, logic LogicFunc) error {
	cfg := config.Cfg.Kafka.Consumer
	batchSize := cfg.BatchSize
	batchTimeout := time.Duration(cfg.BatchTimeout) * time.Millisecond
	batch := make([]*sarama.ConsumerMessage, 0, batchSize)
	ticker := time.NewTicker(batchTimeout)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				if len(batch) > 0 {
					processBatch(session, batch, logic)
				}
				return nil
			}
			batch = append(batch, msg)
			if len(batch) >= batchSize {
				processBatch(session, batch, logic)
				// 清空缓冲区 & 重值定时器
				batch = make([]*sarama.ConsumerMessage, 0, batchSize)
				ticker.Reset(batchTimeout)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				processBatch(session, batch, logic)
				batch = make([]*sarama.ConsumerMessage, 0, batchSize)
			}
		case <-session.Context().Done():
			return nil
		}
	}
}

// processBatch 并发处理一批消息
func processBatch(session sarama.ConsumerGroupSession, messages []*sarama.ConsumerMessage, logic LogicFunc) {
	var wg sync.WaitGroup
	cfg := config.Cfg.Kafka.Consumer

	for i, msg := range messages {
		wg.Add(1)

		go func(idx int, m *sarama.ConsumerMessage) {
			traceID := "job-" + uuid.NewString()
			ctx := context.WithValue(session.Context(), logger.TraceIDKey, traceID)

			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.ErrorContext(ctx, "panic in kafka worker", "recover", r)
				}
			}()

			log.InfoContext(ctx, "kafka message consumer start", "topic", m.Topic, "partition", m.Partition, "offset", m.Offset)

			maxRetries := cfg.MaxRetries
			retryInterval := time.Duration(cfg.RetryInterval) * time.Millisecond

			for retry := 0; ; retry++ {
				err := logic(ctx, m)
				if err == nil {
					log.InfoContext(ctx, "kafka message processed successfully", "message index", idx)
					break
				}

				if maxRetries != -1 && retry >= maxRetries {
					log.ErrorContext(ctx, "reached max retries, dropping message", "err", err, "attempts", retry+1)
					break
				}

				select {
				case <-ctx.Done():
					return
				default:
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(retryInterval):
				}

				retryInterval *= 2
				if retryInterval > 5*time.Second {
					retryInterval = 5 * time.Second
				}
			}
		}(i, msg)
	}

	wg.Wait()

	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		session.MarkMessage(lastMsg, "")
	}
}

// ToCanalMessage 将kafka消息转换为canal消息结构体
func ToCanalMessage(msg *sarama.ConsumerMessage, tableName string) (*CanalMessage, error) {
	var canalMsg CanalMessage
	if err := json.Unmarshal(msg.Value, &canalMsg); err != nil {
		log.Error("unmarshal canal message error", "err", err)
		return nil, err
	}

	if canalMsg.Table != tableName {
		return nil, errors.New("table name not match")
	}

	if len(canalMsg.Data) == 0 {
		return nil, errors.New("data is empty")
	}

	return &canalMsg, nil
}
