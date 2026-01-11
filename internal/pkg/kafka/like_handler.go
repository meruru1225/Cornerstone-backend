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

type LikesHandler struct {
	postRepo   repository.PostRepo
	sysBoxRepo mongo.SysBoxRepo
}

func NewLikesHandler(postRepo repository.PostRepo, sysBox mongo.SysBoxRepo) *LikesHandler {
	return &LikesHandler{
		postRepo:   postRepo,
		sysBoxRepo: sysBox,
	}
}

func (s *LikesHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("post like consumer setup")
	return nil
}

func (s *LikesHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("post like consumer cleanup")
	return nil
}

func (s *LikesHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-like consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-like process batch error", "err", err)
		return err
	}
	return nil
}

func (s *LikesHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// 1. 解析 Canal 消息
	canalMsg, err := ToCanalMessage(msg, "likes")
	if err != nil {
		return err
	}

	// 2. 根据事件类型执行对应操作 (点赞通常是物理增删)
	switch canalMsg.Type {
	case INSERT:
		return s.handleInsert(ctx, canalMsg)
	case DELETE:
		return s.handleDelete(ctx, canalMsg)
	default:
		return nil
	}
}

// handleInsert 处理新增点赞：INCR + DIRTY
func (s *LikesHandler) handleInsert(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	row := msg.Data[0]
	userID, postID := StrToUint64(row["user_id"]), StrToUint64(row["post_id"])

	ExecAction(ctx, ActionParams{
		TargetID:       postID,
		CountKeyPrefix: consts.PostLikeKey,
		DirtyKey:       consts.PostDirtyKey,
		IsIncrement:    true,
		NotifyFunc:     func() { s.sendLikeNotification(ctx, userID, postID) },
	})

	log.InfoContext(ctx, "post like inserted", "userID", userID, "postID", postID)
	return nil
}

// handleDelete 处理取消点赞：DECR + DIRTY
func (s *LikesHandler) handleDelete(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	postID := StrToUint64(msg.Data[0]["post_id"])

	// 接入 ExecAction
	ExecAction(ctx, ActionParams{
		TargetID:       postID,
		CountKeyPrefix: consts.PostLikeKey,
		DirtyKey:       consts.PostDirtyKey,
		IsIncrement:    false,
	})

	log.InfoContext(ctx, "post unlike processed", "postID", postID)
	return nil
}

// sendLikeNotification 封装通知逻辑
func (s *LikesHandler) sendLikeNotification(ctx context.Context, senderID, postID uint64) {
	posts, err := s.postRepo.GetPostByIds(ctx, []uint64{postID})
	if err != nil || len(posts) == 0 {
		log.WarnContext(ctx, "failed to get post for notification", "postID", postID)
		return
	}
	post := posts[0]

	if post.UserID == senderID {
		return
	}

	notification := &mongo.SysBoxModel{
		ReceiverID: post.UserID,
		SenderID:   senderID,
		Type:       1, // 1-帖子点赞
		TargetID:   postID,
		Content:    "点赞了你的帖子",
		Payload: map[string]any{
			"post_title": post.Title,
		},
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := s.sysBoxRepo.CreateNotification(ctx, notification); err != nil {
		log.ErrorContext(ctx, "failed to create like notification", "postID", postID, "err", err)
	}
}
