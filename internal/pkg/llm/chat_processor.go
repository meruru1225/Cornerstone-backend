package llm

import (
	"context"
	log "log/slog"
	"strings"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
)

var mapChatIdToChain = make(map[string]*chains.LLMChain)

func chat(ctx context.Context, content string) (string, error) {
	// TODO: 搜索帖子内容，并入searchChatPrompt

	resp, err := fetchModel(ctx, searchChatPrompt, content, 0.7)
	if err != nil {
		log.Error("AI大模型请求失败", "err", err)
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Content, nil
	}

	return "", nil

}

func chatWithChain(ctx context.Context, question string, chatId string) (chan string, error) {
	split := strings.Split(chatPrompt, "---")
	SystemPromptTpl := split[0]
	UserPromptTpl := split[1]

	promptTemplate := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			SystemPromptTpl,
			nil,
		),

		prompts.NewHumanMessagePromptTemplate(
			UserPromptTpl,
			[]string{"content", "question"},
		),
	})

	chain, ok := mapChatIdToChain[chatId]
	if !ok {
		mem := memory.NewConversationBuffer()
		chain = chains.NewLLMChain(llmClient, promptTemplate)
		chain.Memory = mem
		mapChatIdToChain[chatId] = chain
	}

	// TODO: 搜索帖子内容，并入content
	inputs := map[string]any{
		"content":  nil,
		"question": question,
	}

	stream := make(chan string, 10)

	go func() {
		defer close(stream)

		_, err := chains.Call(
			ctx,
			chain,
			inputs,
			chains.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				stream <- string(chunk)
				return nil
			}),
		)

		if err != nil {
			log.Error("AI大模型请求失败", "err", err)
		}
	}()

	return stream, nil
}
