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

func (h *UserFollowHandler) GetUserFollowers(c *gin.Context) {
	userId := c.GetUint64("user_id")

	limit, offset := h.getPagination(c)

	followers, err := h.userFollowSvc.GetUserFollowers(c, userId, limit, offset)
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	response.Success(c, followers)
}

func (h *UserFollowHandler) GetUserFollowings(c *gin.Context) {
	userId := c.GetUint64("user_id")

	limit, offset := h.getPagination(c)

	followings, err := h.userFollowSvc.GetUserFollowing(c, userId, limit, offset)
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	response.Success(c, followings)
}

func (h *UserFollowHandler) GetUserFollowersCount(c *gin.Context) {
	userId := c.GetUint64("user_id")
	count, err := h.userFollowSvc.GetUserFollowerCount(c, userId)
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	response.Success(c, map[string]int64{"count": count})
}

func (h *UserFollowHandler) GetUserFollowingCount(c *gin.Context) {
	userId := c.GetUint64("user_id")
	count, err := h.userFollowSvc.GetUserFollowingCount(c, userId)
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	response.Success(c, map[string]int64{"count": count})
}

func (h *UserFollowHandler) GetSomeoneIsFollowing(c *gin.Context) {
	userId := c.GetUint64("user_id")
	followingIdStr := c.Query("following_id")

	if followingIdStr == "" {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}

	followingId, err := strconv.ParseUint(followingIdStr, 10, 64)
	if err != nil {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}
	userFollow, err := h.userFollowSvc.GetSomeoneIsFollowing(c, userId, followingId)
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	response.Success(c, userFollow)
}

func (h *UserFollowHandler) Follow(c *gin.Context) {
	userId := c.GetUint64("user_id")
	followingId, err := h.getFollowingId(c)
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	err = h.userFollowSvc.CreateUserFollow(c, &model.UserFollow{
		FollowerID:  userId,
		FollowingID: followingId,
	})
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *UserFollowHandler) Unfollow(c *gin.Context) {
	userId := c.GetUint64("user_id")
	followingId, err := h.getFollowingId(c)
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	err = h.userFollowSvc.DeleteUserFollow(c, &model.UserFollow{
		FollowerID:  userId,
		FollowingID: followingId,
	})
	if err != nil {
		response.ProcessError(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *UserFollowHandler) getFollowingId(c *gin.Context) (uint64, error) {
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

func (h *UserFollowHandler) getPagination(c *gin.Context) (int, int) {
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
