package handler

import (
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"

	"github.com/gin-gonic/gin"
)

type UserContentMetricHandler struct {
	userContentMetricService service.UserContentMetricService
}

func NewUserContentMetricHandler(userContentSvc service.UserContentMetricService) *UserContentMetricHandler {
	return &UserContentMetricHandler{
		userContentMetricService: userContentSvc,
	}
}

// GetMetrics7Days 获取创作者 7 天全作品汇总趋势
func (h *UserContentMetricHandler) GetMetrics7Days(c *gin.Context) {
	userID := c.GetUint64("user_id")

	metricData, err := h.userContentMetricService.GetUserContentMetricsBy7Days(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, metricData)
}

// GetMetrics30Days 获取创作者 30 天全作品汇总趋势
func (h *UserContentMetricHandler) GetMetrics30Days(c *gin.Context) {
	userID := c.GetUint64("user_id")

	metricData, err := h.userContentMetricService.GetUserContentMetricsBy30Days(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, metricData)
}
