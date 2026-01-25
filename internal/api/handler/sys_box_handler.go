package handler

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"context"
	"io"
	log "log/slog"
	"strconv"
	"time"

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
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		pageSize = 10
	}
	userID := c.GetUint64("user_id")

	list, err := h.sysBoxService.GetNotificationList(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, list)
}

// GetUnreadCount 获取未读数
func (h *SysBoxHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetUint64("user_id")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	initialUnread, err := h.sysBoxService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.SSEvent("", initialUnread)
	c.Writer.Flush()

	channelName := consts.SysBoxUnreadNotifyChannel + strconv.FormatUint(userID, 10)
	pubsub := redis.Subscribe(context.Background(), channelName)
	defer func() {
		_ = pubsub.Close()
	}()

	redisCh := pubsub.Channel()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			log.InfoContext(c.Request.Context(), "SSE client disconnected", "userID", userID)
			return false
		case msg := <-redisCh:
			if msg != nil {
				latestUnread, err := h.sysBoxService.GetUnreadCount(c.Request.Context(), userID)
				if err != nil {
					return false
				}
				c.SSEvent("", latestUnread)
				c.Writer.Flush()
				return true
			}
			return false
		case <-time.After(30 * time.Second):
			c.SSEvent("ping", map[string]string{"status": "alive"})
			c.Writer.Flush()
			return true
		}
	})
}

// MarkRead 标记单条已读
func (h *SysBoxHandler) MarkRead(c *gin.Context) {
	var req struct {
		MsgID string `json:"msg_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	userID := c.GetUint64("user_id")
	err := h.sysBoxService.MarkRead(c.Request.Context(), userID, req.MsgID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// MarkAllRead 一键已读
func (h *SysBoxHandler) MarkAllRead(c *gin.Context) {
	userID := c.GetUint64("user_id")
	err := h.sysBoxService.MarkAllRead(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
