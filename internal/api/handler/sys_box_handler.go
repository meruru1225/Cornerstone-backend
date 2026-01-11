package handler

import (
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SysBoxHandler struct {
	sysBoxService service.SysBoxService
}

func NewSysBoxHandler(s service.SysBoxService) *SysBoxHandler {
	return &SysBoxHandler{
		sysBoxService: s,
	}
}

// GetNotificationList 获取通知列表
func (h *SysBoxHandler) GetNotificationList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	userID := c.GetUint64("userID")

	list, err := h.sysBoxService.GetNotificationList(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, list)
}

// GetUnreadCount 获取未读数
func (h *SysBoxHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetUint64("userID")

	unread, err := h.sysBoxService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, unread)
}

// MarkRead 标记单条已读
func (h *SysBoxHandler) MarkRead(c *gin.Context) {
	var req struct {
		MsgID string `json:"msgId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	userID := c.GetUint64("userID")
	err := h.sysBoxService.MarkRead(c.Request.Context(), userID, req.MsgID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// MarkAllRead 一键已读
func (h *SysBoxHandler) MarkAllRead(c *gin.Context) {
	userID := c.GetUint64("userID")
	err := h.sysBoxService.MarkAllRead(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
