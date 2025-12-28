package llm

import (
	"Cornerstone/internal/api/config"
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

func fetchModel(ctx context.Context, systemPrompt string, userPrompt string, temp float64) (*llms.ContentResponse, error) {
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
	return llmClient.GenerateContent(ctx, messages,
		llms.WithModel(config.Cfg.LLM.TextModel),
		llms.WithTemperature(temp),
	)
}

func fetchModelByPicUrls(ctx context.Context, systemPrompt string, picUrls []string, temp float64) (*llms.ContentResponse, error) {
	contentPart := make([]llms.ContentPart, len(picUrls))
	for i, url := range picUrls {
		contentPart[i] = llms.ImageURLPart(url)
	}

	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(systemPrompt),
			},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: contentPart,
		},
	}
	log.Info("正在请求AI大模型")
	return llmClient.GenerateContent(ctx, messages,
		llms.WithModel(config.Cfg.LLM.VisionModel),
		llms.WithTemperature(temp),
	)
}
