package llm

import (
	"context"
	log "log/slog"
	"os"

	"github.com/tmc/langchaingo/llms"
)

func readPrompt(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Error("读取prompt文件失败", "err", err)
		return ""
	}
	return string(data)
}

func fetchModel(ctx context.Context, systemPrompt string, userPrompt string) (*llms.ContentResponse, error) {
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(systemPrompt),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(userPrompt),
			},
		},
	}
	log.Info("正在请求AI大模型")
	return llmClient.GenerateContent(ctx, messages)
}
