package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/mongo"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"
	"strconv"
	"time"

	"github.com/IBM/sarama"
)

type CommentLikesHandler struct {
	actionRepo repository.PostActionRepo
	sysBoxRepo mongo.SysBoxRepo
}

func NewCommentLikesHandler(actionRepo repository.PostActionRepo, sysBox mongo.SysBoxRepo) *CommentLikesHandler {
	return &CommentLikesHandler{
		actionRepo: actionRepo,
		sysBoxRepo: sysBox,
	}
}

func (s *CommentLikesHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("comment like consumer setup")
	return nil
}

func (s *CommentLikesHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("comment like consumer cleanup")
	return nil
}

func (s *CommentLikesHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-comment-like consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-comment-like process batch error", "err", err)
		return err
	}
	return nil
}

func (s *CommentLikesHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	canalMsg, err := ToCanalMessage(msg, "comment_likes")
	if err != nil {
		return err
	}

	switch canalMsg.Type {
	case INSERT:
		return s.handleInsert(ctx, canalMsg)
	case DELETE:
		return s.handleDelete(ctx, canalMsg)
	default:
		return nil
	}
}

// handleInsert: Redis INCR + Dirty Set + SysBox 通知
func (s *CommentLikesHandler) handleInsert(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	row := msg.Data[0]
	userID, commentID := StrToUint64(row["user_id"]), StrToUint64(row["comment_id"])

	ExecAction(ctx, ActionParams{
		TargetID:       commentID,
		CountKeyPrefix: consts.PostCommentLikeKey,
		DirtyKey:       consts.PostCommentLikeDirtyKey,
		IsIncrement:    true,
		NotifyFunc:     func() { s.sendCommentLikeNotification(ctx, userID, commentID) },
	})

	userSetKey := consts.PostCommentLikeUserSetKey + strconv.FormatUint(commentID, 10)
	if err := redis.SAdd(ctx, userSetKey, userID); err != nil {
		return err
	}
	if err := redis.Expire(ctx, userSetKey, 7*24*time.Hour); err != nil {
		return err
	}

	return nil
}

// handleDelete 处理取消评论点赞
func (s *CommentLikesHandler) handleDelete(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	row := msg.Data[0]
	userID, commentID := StrToUint64(row["user_id"]), StrToUint64(row["comment_id"])

	ExecAction(ctx, ActionParams{
		TargetID:       commentID,
		CountKeyPrefix: consts.PostCommentLikeKey,
		DirtyKey:       consts.PostCommentLikeDirtyKey,
		IsIncrement:    false,
	})

	userSetKey := consts.PostCommentLikeUserSetKey + strconv.FormatUint(commentID, 10)
	if err := redis.SRem(ctx, userSetKey, userID); err != nil {
		return err
	}
	return nil
}

// sendCommentLikeNotification 通知逻辑
func (s *CommentLikesHandler) sendCommentLikeNotification(ctx context.Context, senderID, commentID uint64) {
	comment, err := s.actionRepo.GetCommentByID(ctx, commentID)
	if err != nil || comment == nil {
		return
	}

	if comment.UserID == senderID {
		return
	}

	notification := &mongo.SysBoxModel{
		ReceiverID: comment.UserID,
		SenderID:   senderID,
		Type:       4, // 4-评论点赞
		TargetID:   comment.PostID,
		Content:    comment.Content,
		Payload: map[string]any{
			"comment_id": commentID,
		},
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := s.sysBoxRepo.CreateNotification(ctx, notification); err != nil {
		log.ErrorContext(ctx, "failed to create comment-like notification", "err", err)
		return
	}

	// 发布未读数更新通知到 Redis
	channelName := consts.SysBoxUnreadNotifyChannel + strconv.FormatUint(comment.UserID, 10)
	if err = PublishUnreadCountUpdate(ctx, channelName, comment.UserID); err != nil {
		log.ErrorContext(ctx, "failed to publish unread count update", "receiverID", comment.UserID, "err", err)
	}
}
