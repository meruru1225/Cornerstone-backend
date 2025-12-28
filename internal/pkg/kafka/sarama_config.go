package kafka

import (
	"Cornerstone/internal/api/config"
	"time"

	"github.com/IBM/sarama"
)

// newSaramaConfig 是一个包内私有的辅助函数
// 负责统一初始化 sarama.Config，避免代码重复
func newSaramaConfig(kafkaCfg config.KafkaConfig) *sarama.Config {
	c := sarama.NewConfig()

	if kafkaCfg.Sasl.Enable {
		c.Net.SASL.Enable = true
		c.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		c.Net.SASL.User = kafkaCfg.Sasl.Username
		c.Net.SASL.Password = kafkaCfg.Sasl.Password
	}

	c.Consumer.Return.Errors = true
	c.Consumer.Offsets.Initial = sarama.OffsetNewest

	c.Consumer.Group.Session.Timeout = time.Duration(kafkaCfg.Consumer.SessionTimeout) * time.Second
	c.Consumer.Group.Heartbeat.Interval = time.Duration(kafkaCfg.Consumer.HeartbeatInterval) * time.Second
	c.Consumer.Group.Rebalance.Timeout = time.Duration(kafkaCfg.Consumer.RebalanceTimeout) * time.Second
	c.Consumer.Offsets.AutoCommit.Enable = false
	c.Consumer.MaxProcessingTime = time.Duration(kafkaCfg.Consumer.MaxProcessingTime) * time.Second

	return c
}
