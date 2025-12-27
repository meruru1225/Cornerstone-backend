package llm

import (
	"context"
	log "log/slog"
	"strings"

	"github.com/goccy/go-json"
)

const (
	ContentSafePass = iota
	ContentSafeDeny
	ContentSafeWarn

	ContentSafePassStr = "0"
	ContentSafeDenyStr = "1"
	ContentSafeWarnStr = "2"

	ContentSensitive = "sensitive"
)

var mapContentSafe = map[string]int{
	ContentSafePassStr: ContentSafePass,
	ContentSafeDenyStr: ContentSafeDeny,
	ContentSafeWarnStr: ContentSafeWarn,
}

func ContentSafe(ctx context.Context, post *Post) (int, error) {
	postJson, err := json.Marshal(post)
	if err != nil {
		log.Error("AI大模型请求数据序列化失败", "err", err)
		return ContentSafeWarn, err
	}

	resp, err := fetchModel(ctx, contentSafePrompt, string(postJson))

	if err != nil {
		log.Error("AI大模型请求失败", "err", err)
		return ContentSafeWarn, err
	}

	log.Info("AI大模型请求成功", "resp", resp)

	if len(resp.Choices) > 0 {
		if resp.Choices[0].StopReason == ContentSensitive {
			return ContentSafeDeny, nil
		}

		return mapContentSafe[resp.Choices[0].Content], nil
	}

	return ContentSafeWarn, nil
}

func ContentClassify(ctx context.Context, post *Post) (*ClassifyMessage, error) {
	postJson, err := json.Marshal(post)
	if err != nil {
		log.Error("AI大模型请求数据序列化失败", "err", err)
		return nil, err
	}

	resp, err := fetchModel(ctx, contentClassifyPrompt, string(postJson))

	if err != nil {
		log.Error("AI大模型请求失败", "err", err)
		return nil, err
	}

	log.Info("AI大模型请求成功", "resp", resp)

	if len(resp.Choices) > 0 {
		classifyMessage := &ClassifyMessage{}
		cleaned := resp.Choices[0].Content
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		err = json.Unmarshal([]byte(cleaned), classifyMessage)
		if err != nil {
			log.Error("AI大模型返回数据解析失败", "err", err)
			return nil, err
		}

		return classifyMessage, nil
	}

	return nil, nil
}
