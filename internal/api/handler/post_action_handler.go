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
	actionSvc service.PostActionService
}

func NewPostActionHandler(actionSvc service.PostActionService) *PostActionHandler {
	return &PostActionHandler{
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

// GetPostActionState 获取帖子详情页的全量交互状态并上报浏览
func (s *PostActionHandler) GetPostActionState(c *gin.Context) {
	userID := c.GetUint64("user_id")
	postID, err := strconv.ParseUint(c.Param("post_id"), 10, 64)
	if err != nil || postID == 0 {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	ctx := c.Request.Context()
	state := &dto.PostActionStateDTO{}
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		state.LikeCount, _ = s.actionSvc.GetPostLikeCount(gCtx, postID)
		return nil
	})
	g.Go(func() error {
		state.CollectCount, _ = s.actionSvc.GetPostCollectionCount(gCtx, postID)
		return nil
	})
	g.Go(func() error {
		state.CommentCount, _ = s.actionSvc.GetPostCommentCount(gCtx, postID)
		return nil
	})
	g.Go(func() error {
		state.ViewCount, _ = s.actionSvc.GetPostViewCount(gCtx, postID)
		return nil
	})
	g.Go(func() error {
		if userID > 0 {
			state.IsLiked, _ = s.actionSvc.IsLiked(gCtx, userID, postID)
		}
		return nil
	})
	g.Go(func() error {
		if userID > 0 {
			state.IsCollected, _ = s.actionSvc.IsCollected(gCtx, userID, postID)
		}
		return nil
	})

	_ = g.Wait()

	go func() {
		_ = s.actionSvc.TrackPostView(c.Request.Context(), userID, postID)
	}()

	response.Success(c, state)
}

// GetBatchLikes 批量获取点赞数（用于瀑布流渲染）
func (s *PostActionHandler) GetBatchLikes(c *gin.Context) {
	var req dto.PostBatchLikesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	counts, err := s.actionSvc.GetPostLikeCounts(c.Request.Context(), req.PostIDs)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 组装 map 返回，方便前端根据 post_id 直接取值
	res := make(map[uint64]int64)
	for i, pid := range req.PostIDs {
		res[pid] = counts[i]
	}

	response.Success(c, dto.PostBatchLikesDTO{Likes: res})
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
		return
	}
	response.Success(c, nil)
}
