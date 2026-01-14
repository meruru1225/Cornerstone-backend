package handler

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserFollowHandler struct {
	userFollowSvc service.UserFollowService
}

func NewUserFollowHandler(userFollowSvc service.UserFollowService) *UserFollowHandler {
	return &UserFollowHandler{userFollowSvc: userFollowSvc}
}

func (s *UserFollowHandler) GetUserFollowers(c *gin.Context) {
	userId := c.GetUint64("user_id")

	limit, offset := s.getPagination(c)

	followers, err := s.userFollowSvc.GetUserFollowers(c, userId, limit, offset)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, followers)
}

func (s *UserFollowHandler) GetUserFollowings(c *gin.Context) {
	userId := c.GetUint64("user_id")

	limit, offset := s.getPagination(c)

	followings, err := s.userFollowSvc.GetUserFollowing(c, userId, limit, offset)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, followings)
}

func (s *UserFollowHandler) GetUserFollowersCount(c *gin.Context) {
	userId := c.GetUint64("user_id")
	count, err := s.userFollowSvc.GetUserFollowerCount(c, userId)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, map[string]int64{"count": count})
}

func (s *UserFollowHandler) GetUserFollowingCount(c *gin.Context) {
	userId := c.GetUint64("user_id")
	count, err := s.userFollowSvc.GetUserFollowingCount(c, userId)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, map[string]int64{"count": count})
}

func (s *UserFollowHandler) GetSomeoneIsFollowing(c *gin.Context) {
	userId := c.GetUint64("user_id")
	followingIdStr := c.Param("following_id")

	if followingIdStr == "" {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}

	followingId, err := strconv.ParseUint(followingIdStr, 10, 64)
	if err != nil {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}
	userFollow, err := s.userFollowSvc.GetSomeoneIsFollowing(c, userId, followingId)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, userFollow)
}

func (s *UserFollowHandler) Follow(c *gin.Context) {
	userId := c.GetUint64("user_id")
	followingId, err := s.getFollowingId(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userFollowSvc.CreateUserFollow(c, &model.UserFollow{
		FollowerID:  userId,
		FollowingID: followingId,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserFollowHandler) Unfollow(c *gin.Context) {
	userId := c.GetUint64("user_id")
	followingId, err := s.getFollowingId(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	err = s.userFollowSvc.DeleteUserFollow(c, &model.UserFollow{
		FollowerID:  userId,
		FollowingID: followingId,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

func (s *UserFollowHandler) getFollowingId(c *gin.Context) (uint64, error) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		return 0, service.ErrParamInvalid
	}

	val, ok := body["following_id"]
	if !ok {
		return 0, service.ErrParamInvalid
	}

	var followingId uint64
	switch v := val.(type) {
	case float64:
		followingId = uint64(v)
	case string:
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, service.ErrParamInvalid
		}
		followingId = id
	default:
		return 0, service.ErrParamInvalid
	}
	return followingId, nil
}

func (s *UserFollowHandler) getPagination(c *gin.Context) (int, int) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}
	return limit, offset
}
