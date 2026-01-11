package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"context"
	log "log/slog"

	"github.com/IBM/sarama"
)

type ViewsHandler struct {
	// 阅读量逻辑目前主要依赖 Redis 和 ES 标记，暂不需要 Repo 补全数据
}

func NewViewsHandler() *ViewsHandler {
	return &ViewsHandler{}
}

func (s *ViewsHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("post view consumer setup")
	return nil
}

func (s *ViewsHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("post view consumer cleanup")
	return nil
}

func (s *ViewsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-view consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-view process batch error", "err", err)
		return err
	}
	return nil
}

func (s *ViewsHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// 1. 解析 Canal 消息
	canalMsg, err := ToCanalMessage(msg, "post_views")
	if err != nil {
		return err
	}

	// 2. 阅读量通常只有 INSERT (用户阅读)
	// 即使有 DELETE，也只是维护计数平衡
	switch canalMsg.Type {
	case INSERT:
		return s.handleInsert(ctx, canalMsg)
	case DELETE:
		return s.handleDelete(ctx, canalMsg)
	default:
		return nil
	}
}

// handleInsert 处理新增阅读：INCR + DIRTY
func (s *ViewsHandler) handleInsert(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	ExecAction(ctx, ActionParams{
		TargetID:       StrToUint64(msg.Data[0]["post_id"]),
		CountKeyPrefix: consts.PostViewKey,
		DirtyKey:       consts.PostDirtyKey,
		IsIncrement:    true,
	})
	return nil
}

// handleDelete 处理阅读记录删除
func (s *ViewsHandler) handleDelete(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	postID := StrToUint64(msg.Data[0]["post_id"])

	ExecAction(ctx, ActionParams{
		TargetID:       postID,
		CountKeyPrefix: consts.PostViewKey,
		DirtyKey:       consts.PostDirtyKey,
		IsIncrement:    false,
	})

	log.InfoContext(ctx, "post view record deleted", "postID", postID)
	return nil
}
