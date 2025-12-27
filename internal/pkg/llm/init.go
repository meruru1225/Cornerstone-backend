package llm

import (
	"Cornerstone/internal/api/config"
	log "log/slog"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var llmClient llms.Model

var contentSafePrompt string
var contentClassifyPrompt string
var searchChatPrompt string
var chatPrompt string

func InitLLM() error {
	cfg := config.Cfg.LLM

	llm, err := openai.New(
		openai.WithModel(cfg.Model),
		openai.WithToken(cfg.ApiKey),
		openai.WithBaseURL(cfg.URL),
	)

	if err != nil {
		log.Error("AI大模型初始化失败", "err", err)
		return err
	}

	llmClient = llm

	// 从prompt txt文件中读取prompt
	contentSafePrompt = readPrompt("./prompts/content-safe.txt")
	contentClassifyPrompt = readPrompt("./prompts/content-classify.txt")
	searchChatPrompt = readPrompt("./prompts/search-chat.txt")
	chatPrompt = readPrompt("./prompts/chat.txt")

	return nil
}
