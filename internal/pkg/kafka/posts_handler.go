package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/repository"
	"context"
	"fmt"
	log "log/slog"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type PostsHandler struct {
	userDBRepo repository.UserRepo
	postDBRepo repository.PostRepo
	postESRepo es.PostRepo
}

func NewPostsHandler(userDBRepo repository.UserRepo, postDBRepo repository.PostRepo, postESRepo es.PostRepo) *PostsHandler {
	return &PostsHandler{
		userDBRepo: userDBRepo,
		postDBRepo: postDBRepo,
		postESRepo: postESRepo,
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

	toLLMContent := &llm.Post{
		Title:   post.Title,
		Content: post.Content,
	}

	// LLM 审查， 修改审核状态
	safe, err := llm.ContentSafe(ctx, toLLMContent)
	if err != nil {
		return err
	}
	err = s.postDBRepo.UpdatePostStatus(ctx, post.ID, safe)
	if err != nil {
		return err
	}

	// LLM 获取标签
	classify, err := llm.ContentClassify(ctx, toLLMContent)
	if err != nil {
		return err
	}
	if classify != nil {
		if classify.MainTag != "" {
			err = s.postDBRepo.UpsertPostTag(ctx, post.ID, classify.MainTag)
			if err != nil {
				return err
			}
		}
		if len(classify.Tags) > 0 {
			post.AITags = classify.Tags
		}
	}

	// 获取用户设定的标签
	tags := util.ExtractTags(post.Content)
	if len(tags) > 0 {
		post.UserTags = tags
	}

	// 最后导入用户信息并覆写ES
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

	users, err := s.userDBRepo.GetUserSimpleInfoByIds(ctx, []uint64{StrToUint64(msg.Key)})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return errors.New("user not found")
	}
	post.UserNickname = users[0].Nickname
	post.UserAvatar = users[0].AvatarURL
	err = s.postESRepo.IndexPost(ctx, post, canalMsg.TS)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostsHandler) toESModel(message *CanalMessage) (*es.PostES, error) {
	if len(message.Data) == 0 {
		return nil, fmt.Errorf("canal message data is empty")
	}

	row := message.Data[0]

	if row["is_deleted"] == "1" {
		return nil, nil
	}

	return &es.PostES{
		ID:            StrToUint64(row["id"]),
		UserID:        StrToUint64(row["user_id"]),
		Status:        StrToInt(row["status"]),
		Title:         StrToString(row["title"]),
		Content:       StrToString(row["content"]),
		CreatedAt:     StrToDateTime(row["created_at"]),
		UpdatedAt:     StrToDateTime(row["updated_at"]),
		LikesCount:    StrToInt(row["likes_count"]),
		CommentsCount: StrToInt(row["comments_count"]),
		CollectsCount: StrToInt(row["collects_count"]),
	}, nil
}
