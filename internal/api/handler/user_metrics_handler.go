package handler

import (
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"

	"github.com/gin-gonic/gin"
)

type UserMetricsHandler struct {
	userMetricsSvc service.UserMetricsService
}

func NewUserMetricsHandler(userMetricsSvc service.UserMetricsService) *UserMetricsHandler {
	return &UserMetricsHandler{
		userMetricsSvc: userMetricsSvc,
	}
}

func (s *UserMetricsHandler) GetMetrics7Days(c *gin.Context) {
	userID := c.GetUint64("user_id")
	metricData, err := s.userMetricsSvc.GetUserMetricsBy7Days(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, metricData)
}

func (s *UserMetricsHandler) GetMetrics30Days(c *gin.Context) {
	userID := c.GetUint64("user_id")
	metricData, err := s.userMetricsSvc.GetUserMetricsBy30Days(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, metricData)
}
