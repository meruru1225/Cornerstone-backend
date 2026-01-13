package llm

import (
	"Cornerstone/internal/pkg/mongo"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"golang.org/x/sync/errgroup"
)

var tools = []llms.Tool{
	DefineGeneralSearchTool(),
}

type Agent interface {
	ChatSingle(ctx context.Context, userInput string) chan string
	Converse(ctx context.Context, question string, chatId string) chan string
}

type AgentImpl struct {
	handler          *ToolHandler
	agentMessageRepo mongo.AgentMessageRepo
}

func NewAgent(handler *ToolHandler, agentMessageRepo mongo.AgentMessageRepo) Agent {
	return &AgentImpl{
		handler:          handler,
		agentMessageRepo: agentMessageRepo,
	}
}

// ChatSingle 单轮对话Agent
func (s *AgentImpl) ChatSingle(ctx context.Context, userInput string) chan string {
	out := make(chan string, 20)

	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(searchPrompt),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(userInput),
			},
		},
	}

	go func() {
		defer close(out)

		err := s.runAgentLoopStream(ctx, messages, out, 5)
		if err != nil {
			out <- fmt.Sprintf("\n\n> ⚠️ **系统错误**: %v", err)
		}
	}()

	return out
}

// Converse 多轮对话Agent
func (s *AgentImpl) Converse(ctx context.Context, question string, chatId string) chan string {
	out := make(chan string, 100)

	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
		userID = 1
	}

	persistenceCtx := context.Background()

	go func() {
		defer close(out)

		history, _ := s.agentMessageRepo.GetHistory(ctx, chatId, 20)

		var messages []llms.MessageContent
		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart(searchPrompt)},
		})

		// 转换历史记录
		for _, msg := range history {
			role := llms.ChatMessageTypeHuman
			if msg.SenderID == 0 { // 0 约定为 AI
				role = llms.ChatMessageTypeAI
			}
			messages = append(messages, llms.MessageContent{
				Role:  role,
				Parts: []llms.ContentPart{llms.TextPart(msg.Content)},
			})
		}

		// 保存用户当前问题
		_ = s.agentMessageRepo.SaveMessage(persistenceCtx, &mongo.AgentMessage{
			ConversationID: chatId,
			SenderID:       userID,
			Content:        question,
			CreatedAt:      time.Now(),
		})

		// 将当前问题加入对话上下文
		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(question)},
		})

		var aiFullContent strings.Builder
		streamCapture := func(c context.Context, chunk []byte) error {
			str := string(chunk)
			if strings.HasPrefix(str, "[{") || strings.Contains(str, "\"tool_calls\"") {
				return nil
			}
			aiFullContent.WriteString(str)
			out <- str
			return nil
		}

		_ = s.runAgentLoopWithCapture(ctx, messages, out, streamCapture, 5)

		if aiFullContent.Len() > 0 {
			_ = s.agentMessageRepo.SaveMessage(persistenceCtx, &mongo.AgentMessage{
				ConversationID: chatId,
				SenderID:       0, // 0 代表 AI
				Content:        aiFullContent.String(),
				CreatedAt:      time.Now(),
			})
		}
	}()

	return out
}

// runAgentLoop 封装了通用的 ReAct 循环逻辑
func (s *AgentImpl) runAgentLoop(ctx context.Context, messages []llms.MessageContent, maxIter int) (string, error) {
	for i := 0; i < maxIter; i++ {
		// 调用模型决策
		resp, err := fetchAgentCall(ctx, messages, tools, 0.7, false, nil)
		if err != nil {
			return "", err
		}

		choice := resp.Choices[0]

		// 模型决定直接回复文本
		if len(choice.ToolCalls) == 0 {
			if choice.Content != "" {
				return choice.Content, nil
			}
			continue
		}

		// 模型决定调用工具 - 记录模型意图
		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: s.convertToolCallsToParts(choice.ToolCalls),
		})

		// 并行执行工具并同步响应
		toolResponses, err := s.executeTools(ctx, choice.ToolCalls)
		if err != nil {
			return "", err
		}

		// 将工具结果反馈给上下文，进入下一轮迭代
		messages = append(messages, toolResponses...)
	}
	return "抱歉，由于检索轮次过多，我无法在安全时间内为您总结结果。", nil
}

// runAgentLoopStream 将推理过程中的文本和工具状态实时推向 out 通道
func (s *AgentImpl) runAgentLoopStream(ctx context.Context, messages []llms.MessageContent, out chan string, maxIter int) error {
	for i := 0; i < maxIter; i++ {
		var contentBuffer strings.Builder

		streamFunc := func(ctx context.Context, chunk []byte) error {
			str := string(chunk)
			if strings.HasPrefix(str, "[{") || strings.Contains(str, "\"tool_calls\"") {
				return nil
			}
			contentBuffer.WriteString(str)
			out <- str
			return nil
		}

		resp, err := fetchAgentCall(ctx, messages, tools, 0.7, false, streamFunc)
		if err != nil {
			return err
		}

		choice := resp.Choices[0]

		// 模型决定直接回复文本
		if len(choice.ToolCalls) == 0 {
			if contentBuffer.Len() > 0 || choice.Content != "" {
				return nil
			}
			continue
		}

		// 模型决定调用工具 - 向用户同步动作
		for _, tc := range choice.ToolCalls {
			out <- fmt.Sprintf("\n\n> 🛠️ **系统正在执行**: `%s` ...\n\n", tc.FunctionCall.Name)
		}

		// 模型决定调用工具 - 记录模型意图
		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: s.convertToolCallsToParts(choice.ToolCalls),
		})

		// 并行执行工具，并同步响应
		toolMsgs, err := s.executeTools(ctx, choice.ToolCalls)
		if err != nil {
			return err
		}
		messages = append(messages, toolMsgs...)
	}
	out <- "\n\n抱歉，由于检索轮次过多，我无法在安全时间内为您总结结果。"
	return nil
}

// runAgentLoopWithCapture 封装了通用的 ReAct 循环逻辑，支持捕获中间状态
func (s *AgentImpl) runAgentLoopWithCapture(ctx context.Context, messages []llms.MessageContent, out chan string, customStream func(context.Context, []byte) error, maxIter int) error {
	for i := 0; i < maxIter; i++ {
		resp, err := fetchAgentCall(ctx, messages, tools, 0.7, false, customStream)
		if err != nil {
			return err
		}

		choice := resp.Choices[0]

		if len(choice.ToolCalls) == 0 {
			return nil
		}

		// 向用户同步动作
		for _, tc := range choice.ToolCalls {
			out <- fmt.Sprintf("\n\n> 🛠️ **正在调取工具**: `%s` ...\n\n", tc.FunctionCall.Name)
		}

		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: s.convertToolCallsToParts(choice.ToolCalls),
		})

		toolMsgs, err := s.executeTools(ctx, choice.ToolCalls)
		if err != nil {
			return err
		}
		messages = append(messages, toolMsgs...)
	}
	return fmt.Errorf("达到最大迭代次数")
}

// ExecuteTools 通用的并行工具执行器
func (s *AgentImpl) executeTools(ctx context.Context, toolCalls []llms.ToolCall) ([]llms.MessageContent, error) {
	g, gCtx := errgroup.WithContext(ctx)
	toolResponses := make([]llms.ContentPart, len(toolCalls))

	for idx, tc := range toolCalls {
		i, toolCall := idx, tc
		g.Go(func() error {
			handler := s.handler.GetHandleFunction(toolCall.FunctionCall.Name)
			if handler == nil {
				return fmt.Errorf("未定义的工具: %s", toolCall.FunctionCall.Name)
			}

			// 执行具体工具逻辑
			result, err := handler(gCtx, toolCall.FunctionCall.Arguments)
			if err != nil {
				result = fmt.Sprintf("执行失败: %v", err)
			}

			toolResponses[i] = llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    result,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var msgs []llms.MessageContent
	for _, tr := range toolResponses {
		msgs = append(msgs, llms.MessageContent{
			Role:  llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{tr},
		})
	}
	return msgs, nil
}

// convertToolCallsToParts 将工具调用转换为 ContentPart
func (s *AgentImpl) convertToolCallsToParts(tcs []llms.ToolCall) []llms.ContentPart {
	parts := make([]llms.ContentPart, len(tcs))
	for i, tc := range tcs {
		parts[i] = tc
	}
	return parts
}
