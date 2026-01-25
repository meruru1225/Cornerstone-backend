package kafka

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	"fmt"
	log "log/slog"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type UserDetailHandler struct {
	userDBFollowRepo repository.UserFollowRepo
	userESRepo       es.UserRepo
}

func NewUserDetailHandler(userFollowDBRepo repository.UserFollowRepo, userESRepo es.UserRepo) *UserDetailHandler {
	return &UserDetailHandler{
		userDBFollowRepo: userFollowDBRepo,
		userESRepo:       userESRepo,
	}
}

func (s *UserDetailHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("user consumer setup")
	return nil
}

func (s *UserDetailHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("user consumer cleanup")
	return nil
}

func (s *UserDetailHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-user-detail consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-user-detail process batch error", "err", err)
		return err
	}
	log.Info("topic-user-detail consume claim end")
	return nil
}

func (s *UserDetailHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	canalMsg, err := ToCanalMessage(msg, "user_detail")
	if err != nil {
		return err
	}
	user, err := s.toESModel(canalMsg)
	if err != nil {
		return err
	}
	if canalMsg.Type == UPDATE {
		lockKey := consts.UserDetailESLock + strconv.FormatUint(user.ID, 10)
		uuidStr := uuid.NewString()
		lock, err := redis.TryLock(ctx, lockKey, uuidStr, 30*time.Second, 0)
		defer redis.UnLock(ctx, lockKey, uuidStr)
		if err != nil {
			return err
		}
		if !lock {
			return nil
		}
		exist, err := s.userESRepo.Exist(ctx, user.ID)
		if err != nil {
			return err
		}
		if !exist {
			return nil
		}
		return s.userESRepo.IndexUser(ctx, user, canalMsg.TS)
	} else if canalMsg.Type == INSERT {
		return s.userESRepo.IndexUser(ctx, user, canalMsg.TS)
	}
	return nil
}

func (s *UserDetailHandler) toESModel(message *CanalMessage) (*es.UserES, error) {
	if len(message.Data) == 0 {
		return nil, fmt.Errorf("canal message data is empty")
	}

	row := message.Data[0]

	bio := StrToString(row["bio"])
	model := &es.UserES{
		ID:             StrToUint64(row["user_id"]),
		Nickname:       StrToString(row["nickname"]),
		Bio:            &bio,
		AvatarURL:      StrToString(row["avatar_url"]),
		Gender:         StrToInt(row["gender"]),
		Region:         StrToString(row["region"]),
		Birthday:       StrToDate(row["birthday"]),
		FollowersCount: StrToInt(row["followers_count"]),
		FollowingCount: StrToInt(row["following_count"]),
	}

	return model, nil
}
