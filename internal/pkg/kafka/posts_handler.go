package kafka

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/processor"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	"fmt"
	log "log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type PostsHandler struct {
	userDBRepo       repository.UserRepo
	postDBRepo       repository.PostRepo
	postESRepo       es.PostRepo
	contentProcesser processor.ContentLLMProcessor
}

func NewPostsHandler(userDBRepo repository.UserRepo, postDBRepo repository.PostRepo, postESRepo es.PostRepo, contentProcesser processor.ContentLLMProcessor) *PostsHandler {
	return &PostsHandler{
		userDBRepo:       userDBRepo,
		postDBRepo:       postDBRepo,
		postESRepo:       postESRepo,
		contentProcesser: contentProcesser,
	}
}

func (s *PostsHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("post consumer setup")
	return nil
}

func (s *PostsHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("post consumer cleanup")
	return nil
}

func (s *PostsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-post consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-post process batch error", "err", err)
		return err
	}
	log.Info("topic-post consume claim end")
	return nil
}

func (s *PostsHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	canalMsg, err := ToCanalMessage(msg, "posts")
	if err != nil {
		return err
	}

	post, err := s.toESModel(canalMsg)
	if err != nil {
		return err
	}

	if post == nil {
		return s.postESRepo.DeletePost(ctx, StrToUint64(canalMsg.Data[0]["id"]))
	}

	// 没有内容变更，直接覆写ES
	if !s.checkContentIsChange(canalMsg) {
		getById, err := s.postESRepo.GetPostById(ctx, post.ID)
		if err != nil {
			return err
		}
		if getById != nil {
			post.MainTag = getById.MainTag
			post.UserTags = getById.UserTags
			post.AITags = getById.AITags
			post.ContentVector = getById.ContentVector
			post.AISummary = getById.AISummary
		}
		return s.getUserDetailAndIndexES(ctx, post, canalMsg.TS)
	}

	// LLM 处理文本和媒体
	mediaForLLM := make([]*es.PostMediaES, len(post.Media))
	for i := range post.Media {
		mediaForLLM[i] = &post.Media[i]
	}
	r, err := s.contentProcesser.Process(ctx, post.Title, post.Content, mediaForLLM, false)
	if err != nil {
		return err
	}

	post.Status = int(atomic.LoadInt32(&r.MaxStatus))
	if post.Status == int(llm.ContentSafeDeny) {
		log.WarnContext(ctx, "内容审核未通过，拦截后续处理", "post_id", post.ID)
		if err = s.postDBRepo.UpdatePostStatus(ctx, post.ID, post.Status); err != nil {
			return err
		}
		return s.getUserDetailAndIndexES(ctx, post, canalMsg.TS)
	}

	// LLM 进行语义聚合
	aggress, err := llm.Aggressive(ctx, &llm.TagAggressive{
		MainTags:  r.MainTags,
		Tags:      r.Tags,
		Summaries: r.Summaries,
	})
	if err != nil {
		return err
	} else {
		post.MainTag = aggress.MainTag
		post.AITags = aggress.Tags
		post.AISummary = aggress.Summary
		if aggress.MainTag != "" {
			if err = s.postDBRepo.SyncPostMainTag(ctx, post.ID, aggress.MainTag); err != nil {
				return err
			}
		}
	}

	// LLM 切分向量
	toLLMContent := &llm.Content{
		Title:   &post.Title,
		Content: post.Content,
	}
	vector, err := llm.GetVector(ctx, toLLMContent, aggress.Tags, aggress.Summary)
	if err != nil {
		return err
	} else {
		post.ContentVector = vector
	}

	return s.getUserDetailAndIndexES(ctx, post, canalMsg.TS)
}

func (s *PostsHandler) toESModel(message *CanalMessage) (*es.PostES, error) {
	if len(message.Data) == 0 {
		return nil, fmt.Errorf("canal message data is empty")
	}

	row := message.Data[0]

	if row["is_deleted"] == "1" {
		return nil, nil
	}

	var mediaList []es.PostMediaES
	if val, ok := row["media_list"]; ok && val != nil {
		str, ok := val.(string)
		if !ok {
			str = fmt.Sprint(val)
		}

		if str != "" {
			var dbMedias model.MediaList
			if err := json.Unmarshal([]byte(str), &dbMedias); err != nil {
				return nil, err
			}

			mediaList = make([]es.PostMediaES, 0, len(dbMedias))
			for _, m := range dbMedias {
				mediaList = append(mediaList, es.PostMediaES{
					Type:     m.MimeType,
					URL:      m.MediaURL,
					Cover:    m.CoverURL,
					Width:    m.Width,
					Height:   m.Height,
					Duration: m.Duration,
				})
			}
		}
	}

	return &es.PostES{
		ID:            StrToUint64(row["id"]),
		UserID:        StrToUint64(row["user_id"]),
		Status:        StrToInt(row["status"]),
		Title:         StrToString(row["title"]),
		Content:       StrToString(row["content"]),
		Media:         mediaList,
		CreatedAt:     StrToDateTime(row["created_at"]),
		UpdatedAt:     StrToDateTime(row["updated_at"]),
		LikesCount:    StrToInt(row["likes_count"]),
		CommentsCount: StrToInt(row["comments_count"]),
		CollectsCount: StrToInt(row["collects_count"]),
	}, nil
}

func (s *PostsHandler) getUserDetailAndIndexES(ctx context.Context, post *es.PostES, timeStamp int64) error {
	// 导入用户信息并覆写ES，此处加锁避免并发一致性问题
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	lockKey := consts.UserDetailLock + strconv.FormatUint(post.UserID, 10)
	_, err = redis.TryLock(ctx, lockKey, newUUID.String(), time.Second*5, -1)
	if err != nil {
		return err
	}
	defer redis.UnLock(ctx, lockKey, newUUID.String())

	users, err := s.userDBRepo.GetUserSimpleInfoByIds(ctx, []uint64{post.UserID})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return errors.New("user not found")
	}
	post.UserNickname = users[0].Nickname
	post.UserAvatar = users[0].AvatarURL
	return s.postESRepo.IndexPost(ctx, post, timeStamp)
}

func (s *PostsHandler) checkContentIsChange(message *CanalMessage) bool {
	if message.Type == INSERT {
		return true
	}
	if message.Type == UPDATE && len(message.Old) > 0 {
		row := message.Old[0]
		_, titleChanged := row["title"]
		_, contentChanged := row["content"]
		_, mediaChanged := row["media_list"]
		return titleChanged || contentChanged || mediaChanged
	}
	return false
}
