package llm

import (
	"Cornerstone/internal/api/config"
	log "log/slog"

	"github.com/tmc/langchaingo/llms"
)

// ContentSensitive 定义敏感标签
const ContentSensitive = "sensitive"

var llmClient llms.Model

var (
	aggressiveTagPrompt  string
	chatPrompt           string
	contentProcessPrompt string
	imageProcessPrompt   string
	searchChatPrompt     string
)

func InitLLM() error {
	cfg := config.Cfg.LLM

	// 创建LLM客户端
	llm, err := NewGLMClient(cfg.ApiKey, cfg.URL)
	if err != nil {
		log.Error("LLM Initial Failed", "err", err)
		return err
	}
	llmClient = llm

	// 从prompt txt文件中读取prompt
	promptPath := cfg.PromptsPath
	aggressiveTagPrompt = readPrompt(promptPath.AggressiveTag)
	chatPrompt = readPrompt(promptPath.Chat)
	contentProcessPrompt = readPrompt(promptPath.ContentProcess)
	imageProcessPrompt = readPrompt(promptPath.ImageProcess)
	searchChatPrompt = readPrompt(promptPath.SearchChat)

	log.Info("LLM Initial Success")
	return nil
}
