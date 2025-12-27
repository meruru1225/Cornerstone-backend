package kafka

import (
	"context"
	"errors"
	log "log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/goccy/go-json"
)

const (
	batchSize    = 32
	batchTimeout = 1 * time.Second
)

type LogicFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

// pullMessageBatch 拉取一批消息并执行业务逻辑
func pullMessageBatch(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim, logic LogicFunc) error {
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

	for _, msg := range messages {
		wg.Add(1)

		go func(m *sarama.ConsumerMessage) {
			defer wg.Done()
			var retryInterval = 100 * time.Millisecond

			for {
				err := logic(session.Context(), m)
				if err == nil {
					break
				}
				select {
				case <-session.Context().Done():
					return
				default:
				}

				log.Error("process message error", "err", err)
				time.Sleep(retryInterval)

				retryInterval *= 2
				if retryInterval > 5*time.Second {
					retryInterval = 5 * time.Second
				}
			}
		}(msg)
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
