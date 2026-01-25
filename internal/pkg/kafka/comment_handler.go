package kafka

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/mongo"
	"Cornerstone/internal/pkg/processor"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/goccy/go-json"
)

const (
	CommentStatusApproved int8 = 1
)

type CommentsHandler struct {
	postActionRepo repository.PostActionRepo
	postRepo       repository.PostRepo
	sysBoxRepo     mongo.SysBoxRepo
	processor      processor.ContentLLMProcessor
}

func NewCommentsHandler(
	actionRepo repository.PostActionRepo,
	postRepo repository.PostRepo,
	sysBoxRepo mongo.SysBoxRepo,
	proc processor.ContentLLMProcessor,
) *CommentsHandler {
	return &CommentsHandler{
		postActionRepo: actionRepo,
		postRepo:       postRepo,
		sysBoxRepo:     sysBoxRepo,
		processor:      proc,
	}
}

func (s *CommentsHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("comment audit consumer setup")
	return nil
}

func (s *CommentsHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("comment audit consumer cleanup")
	return nil
}

func (s *CommentsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-comment-audit consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-comment process batch error", "err", err)
		return err
	}
	return nil
}

func (s *CommentsHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	canalMsg, err := ToCanalMessage(msg, "post_comments")
	if err != nil {
		return err
	}

	if canalMsg.Type == UPDATE {
		if s.isSoftDeleted(canalMsg) {
			return s.handleDelete(ctx, canalMsg)
		}
		return nil
	}

	if canalMsg.Type != INSERT {
		return nil
	}

	commentModel, err := s.parseToModel(canalMsg)
	if err != nil || commentModel == nil {
		return err
	}

	// 准备审核素材
	mediaForAudit := make([]*es.PostMediaES, 0, len(commentModel.MediaInfo))
	for _, m := range commentModel.MediaInfo {
		mediaForAudit = append(mediaForAudit, &es.PostMediaES{
			Type: m.MimeType,
			URL:  m.MediaURL,
		})
	}

	// 执行 LLM 审核
	res, err := s.processor.Process(ctx, "", commentModel.Content, mediaForAudit, true)
	if err != nil {
		log.ErrorContext(ctx, "comment processor execution failed", "id", commentModel.ID, "err", err)
		return err
	}

	finalStatus := atomic.LoadInt32(&res.MaxStatus)

	return s.handleAuditResult(ctx, commentModel, int8(finalStatus))
}

// handleAuditResult 处理审核结果
func (s *CommentsHandler) handleAuditResult(ctx context.Context, m *model.PostComment, status int8) error {
	_ = s.postActionRepo.UpdateCommentStatus(ctx, m.ID, status)

	if status == llm.ContentSafePass {
		ExecAction(ctx, ActionParams{
			TargetID:       m.PostID,
			CountKeyPrefix: consts.PostCommentKey,
			DirtyKey:       consts.PostDirtyKey,
			IsIncrement:    true,
			NotifyFunc:     func() { s.sendCommentNotification(ctx, m) },
		})
	}
	return nil
}

// sendCommentNotification 处理通知分发逻辑
func (s *CommentsHandler) sendCommentNotification(ctx context.Context, m *model.PostComment) {
	// 获取帖子信息以确定接收者
	posts, err := s.postRepo.GetPostByIds(ctx, []uint64{m.PostID})
	if err != nil || len(posts) == 0 {
		return
	}
	post := posts[0]

	// 确定通知接收者 (优先给被回复者，其次给帖子作者)
	var receiverID uint64

	if m.ReplyToUserID > 0 {
		receiverID = m.ReplyToUserID
	} else {
		receiverID = post.UserID
	}

	// 如果操作者是自己，则不发通知
	if receiverID == m.UserID {
		return
	}

	notification := &mongo.SysBoxModel{
		ReceiverID: receiverID,
		SenderID:   m.UserID,
		Type:       3, // 3-评论
		TargetID:   m.PostID,
		Content:    m.Content,
		Payload: map[string]any{
			"comment_id": m.ID,
			"post_title": post.Title,
		},
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if err := s.sysBoxRepo.CreateNotification(ctx, notification); err != nil {
		log.ErrorContext(ctx, "failed to create comment notification", "id", m.ID, "err", err)
		return
	}

	// 发布未读数更新通知到 Redis
	channelName := consts.SysBoxUnreadNotifyChannel + strconv.FormatUint(receiverID, 10)
	if err = PublishUnreadCountUpdate(ctx, channelName, receiverID); err != nil {
		log.ErrorContext(ctx, "failed to publish unread count update", "receiverID", receiverID, "err", err)
	}
}

// handleDelete 处理软删除逻辑
func (s *CommentsHandler) handleDelete(ctx context.Context, msg *CanalMessage) error {
	if len(msg.Data) == 0 {
		return nil
	}
	row := msg.Data[0]
	status := int8(StrToInt(row["status"]))

	if status == CommentStatusApproved {
		postID := StrToUint64(row["post_id"])
		commentID := StrToUint64(row["id"])

		ExecAction(ctx, ActionParams{
			TargetID:       postID,
			CountKeyPrefix: consts.PostCommentKey,
			DirtyKey:       consts.PostDirtyKey,
			IsIncrement:    false,
		})

		log.InfoContext(ctx, "comment deleted: DECR and DIRTY set", "id", commentID, "postID", postID)
	}

	return nil
}

func (s *CommentsHandler) isSoftDeleted(msg *CanalMessage) bool {
	if len(msg.Data) == 0 || len(msg.Old) == 0 {
		return false
	}
	oldVal, okOld := msg.Old[0]["is_deleted"]
	newVal, okNew := msg.Data[0]["is_deleted"]
	return okOld && okNew && oldVal == "0" && newVal == "1"
}

func (s *CommentsHandler) parseToModel(msg *CanalMessage) (*model.PostComment, error) {
	if len(msg.Data) == 0 {
		return nil, nil
	}
	row := msg.Data[0]

	comment := &model.PostComment{
		ID:            StrToUint64(row["id"]),
		PostID:        StrToUint64(row["post_id"]),
		UserID:        StrToUint64(row["user_id"]),
		ReplyToUserID: StrToUint64(row["reply_to_user_id"]), // 解析被回复人ID
		Content:       StrToString(row["content"]),
		Status:        int8(StrToInt(row["status"])),
	}

	if val, ok := row["media_info"]; ok && val != nil {
		str, _ := val.(string)
		if str != "" {
			var media model.MediaList
			if err := json.Unmarshal([]byte(str), &media); err == nil {
				comment.MediaInfo = media
			}
		}
	}
	return comment, nil
}
