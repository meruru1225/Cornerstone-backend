package handler

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentResponse struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type AgentHandler struct {
	agent llm.Agent
}

func NewAgentHandler(agent llm.Agent) *AgentHandler {
	return &AgentHandler{agent: agent}
}

func (s *AgentHandler) Search(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	channel := s.agent.ChatSingle(c.Request.Context(), query)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-channel; ok {
			c.SSEvent("", AgentResponse{
				Type:    "message",
				Content: msg,
			})
			return true
		}
		return false
	})
}

func (s *AgentHandler) Converse(c *gin.Context) {
	var convDTO dto.AgentConverseRequest

	if err := c.ShouldBindJSON(&convDTO); err != nil {
		response.Fail(c, response.BadRequest, "参数格式错误")
		return
	}

	if convDTO.Question == "" {
		response.Fail(c, response.BadRequest, "问题不能为空")
		return
	}

	isNewChat := false
	if convDTO.ChatID == "" || convDTO.ChatID == "0" {
		convDTO.ChatID = uuid.NewString()
		isNewChat = true
	}

	outChan := s.agent.Converse(c.Request.Context(), convDTO.Question, convDTO.ChatID)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		if isNewChat {
			c.SSEvent("", AgentResponse{
				Type:    "chat_id",
				Content: convDTO.ChatID,
			})
			isNewChat = false
			return true
		}

		if msg, ok := <-outChan; ok {
			c.SSEvent("", AgentResponse{
				Type:    "message",
				Content: msg,
			})
			return true
		}

		c.SSEvent("", AgentResponse{
			Type:    "done",
			Content: "EOF",
		})
		return false
	})
}
