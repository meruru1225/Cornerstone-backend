package llm

import (
	"Cornerstone/internal/api/config"
	"context"
	"errors"
	log "log/slog"
	"os"

	"github.com/tmc/langchaingo/llms"
)

type StreamFunc func(ctx context.Context, chunk []byte) error

func readPrompt(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Error("读取prompt文件失败", "err", err)
		return ""
	}
	return string(data)
}

func fetchModel(ctx context.Context, systemPrompt string, userPrompt string, temp float64) (*llms.ContentResponse, error) {
	if err := TextSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer TextSem.Release(1)
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
	if err := ImageSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer ImageSem.Release(1)
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

func fetchModelEmbedding(ctx context.Context, s string) ([]float32, error) {
	if err := EmbedSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer EmbedSem.Release(1)

	log.Info("正在请求AI大模型")

	vectors, err := llmClient.CreateEmbedding(ctx, []string{s})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return nil, errors.New("vector is empty")
	}
	return vectors[0], nil
}

func fetchAgentCall(ctx context.Context, messages []llms.MessageContent, tools []llms.Tool, temp float64, withImage bool, streamFunc StreamFunc) (*llms.ContentResponse, error) {
	if err := TextSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer TextSem.Release(1)

	model := config.Cfg.LLM.TextModel
	if withImage {
		model = config.Cfg.LLM.VisionModel
	}

	if streamFunc != nil {
		return llmClient.GenerateContent(ctx, messages,
			llms.WithModel(model),
			llms.WithTemperature(temp),
			llms.WithTools(tools),
			llms.WithStreamingFunc(streamFunc),
		)
	}

	return llmClient.GenerateContent(ctx, messages,
		llms.WithModel(model),
		llms.WithTemperature(temp),
		llms.WithTools(tools),
	)
}
