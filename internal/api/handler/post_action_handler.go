package handler

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type PostActionHandler struct {
	postSvc   service.PostService
	actionSvc service.PostActionService
}

func NewPostActionHandler(postSvc service.PostService, actionSvc service.PostActionService) *PostActionHandler {
	return &PostActionHandler{
		postSvc:   postSvc,
		actionSvc: actionSvc,
	}
}

// LikePost 点赞/取消点赞帖子
func (s *PostActionHandler) LikePost(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("post_id"), 10, 64)
	if err != nil || postID == 0 {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	userID := c.GetUint64("user_id")
	var req dto.PostActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	if req.Action == 1 {
		err = s.actionSvc.LikePost(c.Request.Context(), userID, postID)
	} else {
		err = s.actionSvc.CancelLikePost(c.Request.Context(), userID, postID)
	}

	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// CollectPost 收藏/取消收藏帖子
func (s *PostActionHandler) CollectPost(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("post_id"), 10, 64)
	if err != nil || postID == 0 {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	userID := c.GetUint64("user_id")
	var req dto.PostActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	if req.Action == 1 {
		err = s.actionSvc.CollectPost(c.Request.Context(), userID, postID)
	} else {
		err = s.actionSvc.CancelCollectPost(c.Request.Context(), userID, postID)
	}

	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// GetPostActionState 获取帖子详情页的全量交互状态并选择性上报浏览
func (s *PostActionHandler) GetPostActionState(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		PostID    uint64 `form:"post_id" binding:"required"`
		NeedTrack bool   `form:"need_track"`
	}
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	_, err = s.postSvc.GetPostById(c.Request.Context(), req.PostID)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := c.Request.Context()
	state := &dto.PostActionStateDTO{}
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		state.LikeCount, err = s.actionSvc.GetPostLikeCount(gCtx, req.PostID)
		return err
	})
	g.Go(func() error {
		state.CollectCount, err = s.actionSvc.GetPostCollectionCount(gCtx, req.PostID)
		return err
	})
	g.Go(func() error {
		state.CommentCount, err = s.actionSvc.GetPostCommentCount(gCtx, req.PostID)
		return err
	})
	g.Go(func() error {
		state.ViewCount, err = s.actionSvc.GetPostViewCount(gCtx, req.PostID)
		return err
	})

	if userID > 0 {
		g.Go(func() error {
			state.IsLiked, err = s.actionSvc.IsLiked(gCtx, userID, req.PostID)
			return err
		})
		g.Go(func() error {
			state.IsCollected, err = s.actionSvc.IsCollected(gCtx, userID, req.PostID)
			return err
		})

		if req.NeedTrack {
			g.Go(func() error {
				return s.actionSvc.TrackPostView(gCtx, userID, req.PostID)
			})
		}
	}

	err = g.Wait()
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, state)
}

// GetBatchLikes 批量获取点赞数（用于瀑布流渲染）
func (s *PostActionHandler) GetBatchLikes(c *gin.Context) {
	var req dto.PostBatchLikesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	res, err := s.actionSvc.GetPostLikeStates(c.Request.Context(), req.PostIDs)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, res)
}

// CreateComment 发布评论
func (s *PostActionHandler) CreateComment(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req dto.CommentCreateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
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
	commentID, err := strconv.ParseUint(c.Param("comment_id"), 10, 64)
	if err != nil || commentID == 0 {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	var req dto.PostActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	if req.Action == 1 {
		err = s.actionSvc.LikeComment(c.Request.Context(), userID, commentID)
	} else {
		err = s.actionSvc.CancelLikeComment(c.Request.Context(), userID, commentID)
	}

	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// GetComments 获取帖子的评论列表
func (s *PostActionHandler) GetComments(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("post_id"), 10, 64)
	if err != nil {
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

	comments, err := s.actionSvc.GetCommentsByPostID(c.Request.Context(), postID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, comments)
}

// GetSubComments 获取二级评论详情
func (s *PostActionHandler) GetSubComments(c *gin.Context) {
	rootID, err := strconv.ParseUint(c.Param("root_id"), 10, 64)
	if err != nil {
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

// GetUserLikes 获取我/他点赞的列表
func (s *PostActionHandler) GetUserLikes(c *gin.Context) {
	targetUID, _ := strconv.ParseUint(c.Query("user_id"), 10, 64)
	if targetUID == 0 {
		targetUID = c.GetUint64("user_id") // 默认查自己
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		pageSize = 20
	}

	posts, err := s.actionSvc.GetLikedPosts(c.Request.Context(), targetUID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, posts)
}

// GetUserCollections 获取我收藏的列表
func (s *PostActionHandler) GetUserCollections(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		pageSize = 20
	}

	posts, err := s.actionSvc.GetCollectedPosts(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, posts)
}

func (s *PostActionHandler) ReportPost(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("post_id"), 10, 64)
	if err != nil || postID == 0 {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	var req dto.PostReport
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	if err := s.actionSvc.ReportPost(c.Request.Context(), c.GetUint64("user_id"), postID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}
