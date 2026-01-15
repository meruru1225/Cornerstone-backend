package handler

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type IMHandler struct {
	imService service.IMService
}

func NewIMHandler(imService service.IMService) *IMHandler {
	return &IMHandler{imService: imService}
}

// SendMessage 发送消息接口
func (s *IMHandler) SendMessage(c *gin.Context) {
	var req dto.SendMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	// 从 Context 中获取中间件解析出的当前用户 ID
	senderID := c.GetUint64("user_id")

	res, err := s.imService.SendMessage(c, senderID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// MarkAsRead 标记已读接口
func (s *IMHandler) MarkAsRead(c *gin.Context) {
	var req dto.MarkAsReadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	userID := c.GetUint64("user_id")

	err := s.imService.MarkAsRead(c, userID, req.ConversationID, req.Sequence)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// GetChatHistory 获取历史消息
func (s *IMHandler) GetChatHistory(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Query("conv_id"), 10, 64)
	lastSeq, _ := strconv.ParseUint(c.Query("last_seq"), 10, 64)
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	res, err := s.imService.GetChatHistory(c, convID, lastSeq, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// GetNewMessages 获取新消息接口
func (s *IMHandler) GetNewMessages(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Query("conv_id"), 10, 64)
	lastSeq, _ := strconv.ParseUint(c.Query("last_seq"), 10, 64)
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	res, err := s.imService.SyncMessages(c, convID, lastSeq, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// GetConversationList 获取会话列表
func (s *IMHandler) GetConversationList(c *gin.Context) {
	userID := c.GetUint64("user_id")
	res, err := s.imService.GetConversationList(c, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}
