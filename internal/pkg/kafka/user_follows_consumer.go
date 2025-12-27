package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/service"
	log "log/slog"
	"reflect"
	"strconv"

	"github.com/IBM/sarama"
	"github.com/goccy/go-json"
)

// TODO: 由消费Kafka变成定时任务处理，避免写放大，优化整体架构

type UserFollowsConsumer struct {
	userMetricsService service.UserMetricsService
}

func (c *UserFollowsConsumer) Setup(sarama.ConsumerGroupSession) error {
	log.Info("user follows consumer setup")
	return nil
}

func (c *UserFollowsConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("user follows consumer cleanup")
	return nil
}

func (c *UserFollowsConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		log.Info("consume message", "msg", string(msg.Value))
		c.handleMessage(session, msg)
		session.MarkMessage(msg, "")
	}
	return nil
}

func NewUserFollowsConsumer(userMetricsService service.UserMetricsService) *UserFollowsConsumer {
	return &UserFollowsConsumer{
		userMetricsService: userMetricsService,
	}
}

func (c *UserFollowsConsumer) handleMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	var canalMsg CanalMessage
	if err := json.Unmarshal(msg.Value, &canalMsg); err != nil {
		log.Error("unmarshal canal message error", "err", err)
		return
	}

	if canalMsg.Table != "user_follows" {
		return
	}

	if len(canalMsg.Data) == 0 {
		return
	}

	for _, data := range canalMsg.Data {
		val, ok := data["following_id"]
		if !ok {
			continue
		}

		fID, ok := val.(string)
		if !ok {
			log.Warn("unexpected type for following_id", "type", reflect.TypeOf(val))
			return
		}
		userID, err := strconv.ParseUint(fID, 10, 64)
		if err != nil {
			log.Error("parse following_id error", "err", err)
			return
		}

		switch canalMsg.Type {
		case consts.INSERT:
			if err := c.userMetricsService.AddCountUserMetrics(session.Context(), userID, 1); err != nil {
				log.Error("add metrics error", "err", err)
			}
		case consts.DELETE:
			if err := c.userMetricsService.AddCountUserMetrics(session.Context(), userID, -1); err != nil {
				log.Error("add metrics error", "err", err)
			}
		}
	}
}
