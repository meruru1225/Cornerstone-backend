package handler

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PostActionHandler struct {
	actionSvc service.PostActionService
}

func NewPostActionHandler(actionSvc service.PostActionService) *PostActionHandler {
	return &PostActionHandler{
		actionSvc: actionSvc,
	}
}

// LikePost 点赞/取消点赞帖子
func (s *PostActionHandler) LikePost(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req dto.PostActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	var err error
	if req.Action == 1 {
		err = s.actionSvc.LikePost(c.Request.Context(), userID, req.PostID)
	} else {
		err = s.actionSvc.CancelLikePost(c.Request.Context(), userID, req.PostID)
	}

	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// CollectPost 收藏/取消收藏帖子
func (s *PostActionHandler) CollectPost(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req dto.PostActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	var err error
	if req.Action == 1 {
		err = s.actionSvc.CollectPost(c.Request.Context(), userID, req.PostID)
	} else {
		err = s.actionSvc.CancelCollectPost(c.Request.Context(), userID, req.PostID)
	}

	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// CreateComment 发布评论
func (s *PostActionHandler) CreateComment(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req dto.CommentCreateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	if err := s.actionSvc.CreateComment(c.Request.Context(), userID, &req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// DeleteComment 删除评论
func (s *PostActionHandler) DeleteComment(c *gin.Context) {
	userID := c.GetUint64("user_id")
	commentID, _ := strconv.ParseUint(c.Param("comment_id"), 10, 64)

	if err := s.actionSvc.DeleteComment(c.Request.Context(), userID, commentID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// LikeComment 点赞/取消点赞评论
func (s *PostActionHandler) LikeComment(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req struct {
		CommentID uint64 `json:"comment_id" binding:"required"`
		Action    int    `json:"action" binding:"required,oneof=1 2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	var err error
	if req.Action == 1 {
		err = s.actionSvc.LikeComment(c.Request.Context(), userID, req.CommentID)
	} else {
		err = s.actionSvc.CancelLikeComment(c.Request.Context(), userID, req.CommentID)
	}

	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// GetComments 获取帖子的评论列表（分层结构）
func (s *PostActionHandler) GetComments(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Query("post_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	comments, err := s.actionSvc.GetCommentsByPostID(c.Request.Context(), postID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, comments)
}

// GetSubComments 获取评论的子评论列表
func (s *PostActionHandler) GetSubComments(c *gin.Context) {
	rootID, _ := strconv.ParseUint(c.Query("root_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if rootID == 0 {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	comments, err := s.actionSvc.GetSubComments(c.Request.Context(), rootID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, comments)
}

// GetUserLikes 获取当前用户点赞过的帖子列表
func (s *PostActionHandler) GetUserLikes(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	posts, err := s.actionSvc.GetLikedPosts(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, posts)
}

// GetUserCollections 获取当前用户收藏的帖子列表
func (s *PostActionHandler) GetUserCollections(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	posts, err := s.actionSvc.GetCollectedPosts(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, posts)
}
