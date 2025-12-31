package llm

import (
	"Cornerstone/internal/api/config"
	"Cornerstone/internal/pkg/es"
	"context"
	"fmt"
	log "log/slog"
	"strings"

	"github.com/goccy/go-json"
	"github.com/tmc/langchaingo/llms"
	"golang.org/x/sync/errgroup"
)

type Agent interface {
	Chat(ctx context.Context, userInput string) (string, error)
	ChatWithChain(ctx context.Context, question string, chatId string) (chan string, error)
}

type AgentImpl struct {
	postRepo es.PostRepo
}

func NewAgent(postRepo es.PostRepo) Agent {
	return &AgentImpl{
		postRepo: postRepo,
	}
}

// Chat 智能对话入口：支持多轮工具调用与并发检索
func (s *AgentImpl) Chat(ctx context.Context, userInput string) (string, error) {
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

	// 限制迭代轮次，防止逻辑黑洞
	maxIterations := 5
	for i := 0; i < maxIterations; i++ {
		// 询问模型意图
		resp, err := s.callLLMWithTools(ctx, messages)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		choice := resp.Choices[0]

		// 如果模型给出了文本回答且没有工具调用，直接返回
		if len(choice.ToolCalls) == 0 && choice.Content != "" {
			return choice.Content, nil
		}

		// 处理工具调用分支
		if len(choice.ToolCalls) > 0 {
			// 将 AI 的意图加入历史记录 (Role: AI)
			aiMsg := llms.MessageContent{Role: llms.ChatMessageTypeAI}
			for _, tc := range choice.ToolCalls {
				aiMsg.Parts = append(aiMsg.Parts, llms.ToolCall{
					ID:   tc.ID,
					Type: "function",
					FunctionCall: &llms.FunctionCall{
						Name:      tc.FunctionCall.Name,
						Arguments: tc.FunctionCall.Arguments,
					},
				})
			}
			messages = append(messages, aiMsg)

			// 并发执行搜索工具
			g, gCtx := errgroup.WithContext(ctx)
			toolResponses := make([]llms.ContentPart, len(choice.ToolCalls))

			for idx, tc := range choice.ToolCalls {
				i, toolCall := idx, tc
				g.Go(func() error {
					var args struct {
						Query string `json:"query"`
					}
					if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
						return err
					}

					searchResult, err := s.executeSearchLogic(gCtx, args.Query)
					if err != nil {
						searchResult = "（站内搜索暂时不可用）"
					}

					toolResponses[i] = llms.ToolCallResponse{
						ToolCallID: toolCall.ID,
						Name:       toolCall.FunctionCall.Name,
						Content:    searchResult,
					}
					return nil
				})
			}

			if err := g.Wait(); err != nil {
				return "", err
			}

			// 将工具结果加入历史 (Role: Tool)
			for _, tr := range toolResponses {
				messages = append(messages, llms.MessageContent{
					Role:  llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{tr},
				})
			}
			continue
		}

		if choice.Content != "" {
			return choice.Content, nil
		}
	}

	return "抱歉，由于检索轮次过多，我无法在安全时间内为您总结结果。", nil
}

func (s *AgentImpl) ChatWithChain(ctx context.Context, question string, chatId string) (chan string, error) {
	log.Info("聊天机器人-链式调用", "ctx", ctx, "question", question, "chatId", chatId)
	return nil, nil
}

// 修改后的通用请求方法，支持工具注入
func (s *AgentImpl) callLLMWithTools(ctx context.Context, messages []llms.MessageContent) (*llms.ContentResponse, error) {
	if err := TextSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer TextSem.Release(1)

	return llmClient.GenerateContent(ctx, messages,
		llms.WithModel(config.Cfg.LLM.TextModel),
		llms.WithTemperature(0.7),
		llms.WithTools([]llms.Tool{DefineGeneralSearchTool()}),
	)
}

func (s *AgentImpl) executeSearchLogic(ctx context.Context, query string) (string, error) {
	vector, err := fetchModelEmbedding(ctx, query)
	if err != nil {
		return "", err
	}
	posts, err := s.postRepo.HybridSearch(ctx, query, vector, 0, 10)
	if err != nil {
		return "", err
	}
	if len(posts) == 0 {
		return "未找到任何相关的站内笔记。", nil
	}
	var builder strings.Builder
	builder.WriteString("以下是为你找到的站内相关笔记，请参考：\n\n")

	for i, post := range posts {
		item := fmt.Sprintf("### 笔记 %d\n- **标题**: %s\n- **作者**: %s\n- **内容**: %s\n- **AI总结**: %s\n---\n",
			i+1, post.Title, post.UserNickname, post.Content, post.AISummary)
		builder.WriteString(item)
	}
	return builder.String(), nil
}
