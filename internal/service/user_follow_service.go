package service

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	"strconv"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

const MaxCacheSize = 1000
const MaxFollowingCount = 1000

type UserFollowService interface {
	GetUserFollowers(ctx context.Context, userId uint64, limit, offset int) ([]*model.UserFollow, error)
	GetUserFollowing(ctx context.Context, userId uint64, limit, offset int) ([]*model.UserFollow, error)
	GetUserFollowerCount(ctx context.Context, userId uint64) (int64, error)
	GetUserFollowingCount(ctx context.Context, userId uint64) (int64, error)
	GetSomeoneIsFollowing(ctx context.Context, userId, followingId uint64) (bool, error)
	CreateUserFollow(ctx context.Context, userFollow *model.UserFollow) error
	DeleteUserFollow(ctx context.Context, userFollow *model.UserFollow) error
}

type UserFollowServiceImpl struct {
	userFollowRepo repository.UserFollowRepo
}

func NewUserFollowService(userFollowRepo repository.UserFollowRepo) UserFollowService {
	return &UserFollowServiceImpl{userFollowRepo: userFollowRepo}
}

type fetchListFunc func(ctx context.Context, userId uint64, limit, offset int) ([]*model.UserFollow, error)
type fetchCountFunc func(ctx context.Context, userId uint64) (int64, error)

func (s *UserFollowServiceImpl) GetUserFollowers(ctx context.Context, userId uint64, limit, offset int) ([]*model.UserFollow, error) {
	return s.getFollowListCommon(
		ctx, userId, limit, offset,
		consts.UserFollowerKey,
		true,
		s.userFollowRepo.GetUserFollowers,
	)
}

func (s *UserFollowServiceImpl) GetUserFollowing(ctx context.Context, userId uint64, limit, offset int) ([]*model.UserFollow, error) {
	return s.getFollowListCommon(
		ctx, userId, limit, offset,
		consts.UserFollowingKey,
		false,
		s.userFollowRepo.GetUserFollowing,
	)
}

func (s *UserFollowServiceImpl) GetUserFollowerCount(ctx context.Context, userId uint64) (int64, error) {
	return s.getCountCommon(
		ctx, userId,
		consts.UserFollowerCountKey,
		s.userFollowRepo.GetUserFollowerCount,
	)
}

func (s *UserFollowServiceImpl) GetUserFollowingCount(ctx context.Context, userId uint64) (int64, error) {
	return s.getCountCommon(
		ctx, userId,
		consts.UserFollowingCountKey,
		s.userFollowRepo.GetUserFollowingCount,
	)
}

func (s *UserFollowServiceImpl) GetSomeoneIsFollowing(ctx context.Context, userId, followingId uint64) (bool, error) {
	key := consts.UserFollowingKey + strconv.FormatUint(userId, 10)
	rdb := redis.GetRdbClient()
	res, err := rdb.ZScore(ctx, key, strconv.FormatUint(followingId, 10)).Result()
	if err == nil && res != 0 {
		return true, nil
	}
	userFollow, err := s.userFollowRepo.GetUserFollow(ctx, userId, followingId)
	if err != nil {
		return false, err
	}
	if userFollow != nil {
		return true, nil
	}
	return false, nil
}

func (s *UserFollowServiceImpl) CreateUserFollow(ctx context.Context, userFollow *model.UserFollow) error {
	if userFollow.FollowerID == userFollow.FollowingID {
		return ErrUserFollowSelf
	}

	count, err := s.GetUserFollowingCount(ctx, userFollow.FollowerID)
	if err != nil {
		return err
	}
	if count >= MaxFollowingCount {
		return ErrUserFollowLimit
	}

	isFollowing, err := s.GetSomeoneIsFollowing(ctx, userFollow.FollowerID, userFollow.FollowingID)
	if err != nil {
		return err
	}
	if isFollowing {
		return ErrUserFollowExist
	}

	userFollow.CreatedAt = time.Now()

	err = s.userFollowRepo.CreateUserFollow(ctx, userFollow)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserFollowServiceImpl) DeleteUserFollow(ctx context.Context, userFollow *model.UserFollow) error {
	err := s.userFollowRepo.DeleteUserFollow(ctx, userFollow)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserFollowServiceImpl) getFollowListCommon(
	ctx context.Context,
	userId uint64,
	limit, offset int,
	keyPrefix string,
	isFollowerList bool,
	fetchDB fetchListFunc,
) ([]*model.UserFollow, error) {
	if offset+limit > MaxCacheSize {
		return fetchDB(ctx, userId, limit, offset)
	}

	key := keyPrefix + strconv.FormatUint(userId, 10)
	rdb := redis.GetRdbClient()

	res, err := rdb.ZRevRangeWithScores(ctx, key, int64(offset), int64(offset+limit-1)).Result()
	if err == nil && len(res) != 0 {
		return s.zSetResToUserFollow(userId, res, isFollowerList)
	}

	dbData, err := fetchDB(ctx, userId, MaxCacheSize, 0)
	if err != nil {
		return nil, err
	}
	if len(dbData) == 0 {
		return []*model.UserFollow{}, nil
	}

	go func(data []*model.UserFollow, cacheKey string, isFollower bool) {
		_ = redis.DeleteKey(context.Background(), cacheKey) // 使用 Background context 防止 cancel
		pipe := rdb.Pipeline()
		zMembers := make([]redisv9.Z, 0, len(data))

		for _, item := range data {
			memberID := item.FollowerID
			if !isFollower {
				memberID = item.FollowingID
			}

			zMembers = append(zMembers, redisv9.Z{
				Score:  float64(item.CreatedAt.Unix()),
				Member: memberID,
			})
		}
		pipe.ZAdd(context.Background(), cacheKey, zMembers...)
		pipe.Expire(context.Background(), cacheKey, time.Hour*1)
		_, _ = pipe.Exec(context.Background())
	}(dbData, key, isFollowerList)

	start := offset
	end := offset + limit
	if start >= len(dbData) {
		return []*model.UserFollow{}, nil
	}
	if end > len(dbData) {
		end = len(dbData)
	}

	return dbData[start:end], nil
}

func (s *UserFollowServiceImpl) getCountCommon(
	ctx context.Context,
	userId uint64,
	keyPrefix string,
	fetchDB fetchCountFunc,
) (int64, error) {
	key := keyPrefix + strconv.FormatUint(userId, 10)

	valStr, err := redis.GetValue(ctx, key)
	if err == nil && valStr != "" {
		return strconv.ParseInt(valStr, 10, 64)
	}

	count, err := fetchDB(ctx, userId)
	if err != nil {
		return 0, err
	}

	_ = redis.SetWithExpiration(ctx, key, count, time.Hour*1)
	return count, nil
}

func (s *UserFollowServiceImpl) zSetResToUserFollow(ownerId uint64, res []redisv9.Z, isFollowerList bool) ([]*model.UserFollow, error) {
	userFollows := make([]*model.UserFollow, 0, len(res))
	for _, v := range res {
		id, err := strconv.ParseUint(v.Member.(string), 10, 64)
		if err != nil {
			return nil, err
		}
		createdAt := v.Score

		item := &model.UserFollow{}

		if isFollowerList {
			item.FollowingID = ownerId
			item.FollowerID = id
			item.CreatedAt = time.Unix(int64(createdAt), 0)
		} else {
			item.FollowerID = ownerId
			item.FollowingID = id
			item.CreatedAt = time.Unix(int64(createdAt), 0)
		}
		userFollows = append(userFollows, item)
	}
	return userFollows, nil
}
