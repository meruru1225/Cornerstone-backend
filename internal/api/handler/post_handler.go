package handler

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	postSvc service.PostService
}

func NewPostHandler(postSvc service.PostService) *PostHandler {
	return &PostHandler{
		postSvc: postSvc,
	}
}

func (s *PostHandler) RecommendPost(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var searchDTO dto.PostListDTO
	if err := c.ShouldBindQuery(&searchDTO); err != nil {
		response.Error(c, err)
		return
	}

	posts, err := s.postSvc.RecommendPost(c.Request.Context(), userID, searchDTO.Page, searchDTO.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, posts)
}

func (s *PostHandler) SearchPost(c *gin.Context) {
	var searchDTO dto.PostListDTO

	if err := c.ShouldBindQuery(&searchDTO); err != nil {
		response.Error(c, err)
		return
	}

	posts, err := s.postSvc.SearchPost(c.Request.Context(), searchDTO.Keyword, searchDTO.Page, searchDTO.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, posts)
}

func (s *PostHandler) CreatePost(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req dto.PostBaseDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	if err := util.ValidateDTO(&req); err != nil {
		response.Error(c, err)
		return
	}

	err := s.postSvc.CreatePost(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

func (s *PostHandler) UpdatePostContent(c *gin.Context) {
	userID := c.GetUint64("user_id")
	postIDStr := c.Param("post_id")

	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	var baseDTO dto.PostBaseDTO
	if err = c.ShouldBindJSON(&baseDTO); err != nil {
		response.Error(c, err)
		return
	}
	if err = util.ValidateDTO(&baseDTO); err != nil {
		response.Error(c, err)
		return
	}

	err = s.postSvc.UpdatePostContent(c.Request.Context(), userID, uint64(postID), &baseDTO)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

func (s *PostHandler) DeletePost(c *gin.Context) {
	userID := c.GetUint64("user_id")
	postIDStr := c.Param("post_id")

	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err = s.postSvc.DeletePost(c.Request.Context(), userID, uint64(postID)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

func (s *PostHandler) GetPost(c *gin.Context) {
	userID := c.GetUint64("user_id")
	postIDStr := c.Param("post_id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		response.Error(c, err)
		return
	}

	post, err := s.postSvc.GetPost(c.Request.Context(), userID, uint64(postID))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, post)
}

func (s *PostHandler) GetPostSelf(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var searchDTO dto.PostListDTO
	err := c.ShouldBindQuery(&searchDTO)
	if err != nil {
		response.Error(c, err)
		return
	}

	posts, err := s.postSvc.GetPostSelf(c.Request.Context(), userID, searchDTO.Page, searchDTO.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, posts)
}

// GetPostByUserId 获取指定用户的公开帖子列表
func (s *PostHandler) GetPostByUserId(c *gin.Context) {
	targetUID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	if targetUID == 0 {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		pageSize = 20
	}

	posts, err := s.postSvc.GetPostByUserId(c.Request.Context(), targetUID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, posts)
}

// GetWarningPosts 审核员：获取待审核列表
func (s *PostHandler) GetWarningPosts(c *gin.Context) {
	lastID, err := strconv.ParseUint(c.DefaultQuery("last_id", "0"), 10, 64)
	if err != nil {
		lastID = 0
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		pageSize = 20
	}

	posts, err := s.postSvc.GetWarningPosts(c.Request.Context(), lastID, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, posts)
}

// UpdatePostStatus 审核员：操作帖子（通过、驳回）
func (s *PostHandler) UpdatePostStatus(c *gin.Context) {
	postIDStr := c.Param("post_id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	var req dto.PostUpdateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	if err := util.ValidateDTO(req); err != nil {
		response.Error(c, err)
		return
	}

	if err := s.postSvc.UpdatePostStatus(c.Request.Context(), uint64(postID), req.Status); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}
