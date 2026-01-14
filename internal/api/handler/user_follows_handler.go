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
		response.Error(c, service.ErrParamInvalid)
		return
	}

	followingId, err := strconv.ParseUint(followingIdStr, 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
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
	followingIdStr := c.Param("following_id")
	followingId, err := strconv.ParseUint(followingIdStr, 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
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
	followingIdStr := c.Param("following_id")
	followingId, err := strconv.ParseUint(followingIdStr, 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
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

func (s *UserFollowHandler) getPagination(c *gin.Context) (int, int) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		pageSize = 10
	}
	return page, pageSize
}
