package kafka

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/processor"
	"Cornerstone/internal/repository"
	"context"
	log "log/slog"
	"sync/atomic"

	"github.com/IBM/sarama"
	"github.com/goccy/go-json"
)

type CommentsHandler struct {
	postActionRepo repository.PostActionRepo
	processor      processor.ContentLLMProcessor
}

func NewCommentsHandler(repo repository.PostActionRepo, proc processor.ContentLLMProcessor) *CommentsHandler {
	return &CommentsHandler{
		postActionRepo: repo,
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
			log.InfoContext(ctx, "comment soft deleted, skip audit", "id", s.getCommentID(canalMsg))
			return nil
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

	mediaForAudit := make([]*es.PostMediaES, 0, len(commentModel.MediaInfo))
	for _, m := range commentModel.MediaInfo {
		mediaForAudit = append(mediaForAudit, &es.PostMediaES{
			Type: m.MimeType,
			URL:  m.MediaURL,
		})
	}

	res, err := s.processor.Process(ctx, "", commentModel.Content, mediaForAudit, true)
	if err != nil {
		log.ErrorContext(ctx, "comment processor execution failed", "id", commentModel.ID, "err", err)
		return err
	}

	finalStatus := atomic.LoadInt32(&res.MaxStatus)

	err = s.syncAuditStatus(ctx, commentModel.ID, int(finalStatus))
	if err != nil {
		log.ErrorContext(ctx, "failed to sync comment audit status", "id", commentModel.ID, "err", err)
		return err
	}

	log.InfoContext(ctx, "comment audit completed", "id", commentModel.ID, "status", finalStatus)
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

func (s *CommentsHandler) getCommentID(msg *CanalMessage) uint64 {
	if len(msg.Data) > 0 {
		return StrToUint64(msg.Data[0]["id"])
	}
	return 0
}

func (s *CommentsHandler) parseToModel(msg *CanalMessage) (*model.PostComment, error) {
	if len(msg.Data) == 0 {
		return nil, nil
	}
	row := msg.Data[0]

	comment := &model.PostComment{
		ID:      StrToUint64(row["id"]),
		Content: StrToString(row["content"]),
		Status:  int8(StrToInt(row["status"])),
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

func (s *CommentsHandler) syncAuditStatus(ctx context.Context, id uint64, status int) error {
	pass := true
	if status == int(llm.ContentSafeDeny) {
		pass = false
	}
	return s.postActionRepo.UpdateCommentStatus(ctx, id, pass)
}
