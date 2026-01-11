package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/mongo"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"
	"time"

	"github.com/IBM/sarama"
)

type CollectionsHandler struct {
	postRepo   repository.PostRepo
	sysBoxRepo mongo.SysBoxRepo
}

func NewCollectionsHandler(postRepo repository.PostRepo, sysBox mongo.SysBoxRepo) *CollectionsHandler {
	return &CollectionsHandler{
		postRepo:   postRepo,
		sysBoxRepo: sysBox,
	}
}

func (s *CollectionsHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("post collection consumer setup")
	return nil
}

func (s *CollectionsHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("post collection consumer cleanup")
	return nil
}

func (s *CollectionsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-collection consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-collection process batch error", "err", err)
		return err
	}
	return nil
}

func (s *CollectionsHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// 1. 解析 Canal 消息
	canalMsg, err := ToCanalMessage(msg, "collections")
	if err != nil {
		return err
	}

	// 2. 处理物理增删
	switch canalMsg.Type {
	case INSERT:
		return s.handleInsert(ctx, canalMsg)
	case DELETE:
		return s.handleDelete(ctx, canalMsg)
	default:
		return nil
	}
}

// handleInsert 处理收藏：INCR + DIRTY
func (s *CollectionsHandler) handleInsert(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	row := msg.Data[0]
	userID := StrToUint64(row["user_id"])
	postID := StrToUint64(row["post_id"])

	ExecAction(ctx, ActionParams{
		TargetID:       postID,
		CountKeyPrefix: consts.PostCollectionKey,
		DirtyKey:       consts.PostDirtyKey,
		IsIncrement:    true,
		NotifyFunc:     func() { s.sendCollectionNotification(ctx, userID, postID) },
	})

	log.InfoContext(ctx, "post collection inserted", "userID", userID, "postID", postID)
	return nil
}

// handleDelete 处理取消收藏：DECR + DIRTY
func (s *CollectionsHandler) handleDelete(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	postID := StrToUint64(msg.Data[0]["post_id"])

	ExecAction(ctx, ActionParams{
		TargetID:       postID,
		CountKeyPrefix: consts.PostCollectionKey,
		DirtyKey:       consts.PostDirtyKey,
		IsIncrement:    false,
	})

	log.InfoContext(ctx, "post collection deleted", "postID", postID)
	return nil
}

// sendCollectionNotification 发送收藏通知
func (s *CollectionsHandler) sendCollectionNotification(ctx context.Context, senderID, postID uint64) {
	posts, err := s.postRepo.GetPostByIds(ctx, []uint64{postID})
	if err != nil || len(posts) == 0 {
		return
	}
	post := posts[0]

	if post.UserID == senderID {
		return
	}

	notification := &mongo.SysBoxModel{
		ReceiverID: post.UserID,
		SenderID:   senderID,
		Type:       2, // 2-帖子收藏
		TargetID:   postID,
		Content:    "收藏了你的帖子",
		Payload: map[string]any{
			"post_title": post.Title,
		},
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := s.sysBoxRepo.CreateNotification(ctx, notification); err != nil {
		log.ErrorContext(ctx, "failed to create collection notification", "postID", postID, "err", err)
	}
}
