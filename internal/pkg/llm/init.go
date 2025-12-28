package llm

import (
	"Cornerstone/internal/api/config"
	log "log/slog"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var llmClient llms.Model

var (
	contentSafePrompt     string
	imageSafePrompt       string
	contentClassifyPrompt string
	searchChatPrompt      string
	chatPrompt            string
)

func InitLLM() error {
	cfg := config.Cfg.LLM

	// 创建LLM客户端
	llm, err := openai.New(
		openai.WithModel(cfg.TextModel),
		openai.WithToken(cfg.ApiKey),
		openai.WithBaseURL(cfg.URL),
	)
	if err != nil {
		log.Error("LLM Initial Failed", "err", err)
		return err
	}
	llmClient = llm

	// 从prompt txt文件中读取prompt
	promptPath := cfg.PromptsPath
	chatPrompt = readPrompt(promptPath.Chat)
	contentSafePrompt = readPrompt(promptPath.ContentSafe)
	contentClassifyPrompt = readPrompt(promptPath.ContentClassify)
	imageSafePrompt = readPrompt(promptPath.ImageSafe)
	searchChatPrompt = readPrompt(promptPath.SearchChat)

	log.Info("LLM Initial Success")
	return nil
}
