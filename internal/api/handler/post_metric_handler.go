package handler

import (
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PostMetricHandler struct {
	postMetricSvc service.PostMetricService
}

func NewPostMetricHandler(postMetricSvc service.PostMetricService) *PostMetricHandler {
	return &PostMetricHandler{
		postMetricSvc: postMetricSvc,
	}
}

// GetMetrics7Days 获取帖子 7 天趋势
func (h *PostMetricHandler) GetMetrics7Days(c *gin.Context) {
	userID := c.GetUint64("user_id")
	postIDStr := c.Param("post_id")
	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	metricData, err := h.postMetricSvc.GetPostMetricsBy7Days(c.Request.Context(), postID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, metricData)
}

// GetMetrics30Days 获取帖子 30 天趋势
func (h *PostMetricHandler) GetMetrics30Days(c *gin.Context) {
	userID := c.GetUint64("user_id")
	postIDStr := c.Param("post_id")
	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	metricData, err := h.postMetricSvc.GetPostMetricsBy30Days(c.Request.Context(), postID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, metricData)
}
