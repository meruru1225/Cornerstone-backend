package handler

import (
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AgentResponse struct {
	Message string `json:"message"`
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
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}

	channel := s.agent.ChatSingle(c.Request.Context(), query)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-channel; ok {
			c.SSEvent("message", AgentResponse{Message: msg})
			return true
		}
		return false
	})
}

func (s *AgentHandler) Converse(c *gin.Context) {
	question := c.PostForm("question")
	chatIDStr := c.PostForm("chat_id")

	if question == "" {
		response.Fail(c, response.BadRequest, service.ErrParamInvalid.Error())
		return
	}

	if chatIDStr == "" || chatIDStr == "0" {
		chatIDStr = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	outChan := s.agent.Converse(c.Request.Context(), question, chatIDStr)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-outChan; ok {
			c.SSEvent("message", msg)
			return true
		}
		return false
	})
}
