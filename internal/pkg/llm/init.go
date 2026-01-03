package llm

import (
	"Cornerstone/internal/api/config"
	log "log/slog"

	"github.com/tmc/langchaingo/llms/openai"
)

// ContentSensitive 定义敏感标签
const ContentSensitive = "sensitive"

var llmClient *openai.LLM

var (
	aggressiveTagPrompt    string
	chatPrompt             string
	contentProcessPrompt   string
	contentAuditOnlyPrompt string
	imageProcessPrompt     string
	imageAuditOnlyPrompt   string
	searchPrompt           string
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
	contentAuditOnlyPrompt = readPrompt(promptPath.ContentAuditOnly)
	imageProcessPrompt = readPrompt(promptPath.ImageProcess)
	imageAuditOnlyPrompt = readPrompt(promptPath.ImageAuditOnly)
	searchPrompt = readPrompt(promptPath.Search)

	log.Info("LLM Initial Success")
	return nil
}
