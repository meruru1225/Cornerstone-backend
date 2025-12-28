package llm

import (
	"context"
	log "log/slog"
	"strings"

	"github.com/goccy/go-json"
)

const (
	ContentSafePass = iota + 1
	ContentSafeDeny
	ContentSafeWarn

	ContentSafePassStr = "1"
	ContentSafeDenyStr = "2"
	ContentSafeWarnStr = "3"

	ContentSensitive = "sensitive"
)

var mapContentSafe = map[string]int{
	ContentSafePassStr: ContentSafePass,
	ContentSafeDenyStr: ContentSafeDeny,
	ContentSafeWarnStr: ContentSafeWarn,
}

var setMainTag = map[string]bool{
	"编程开发": true,
	"科技数码": true,
	"互联网":  true,
	"美食探店": true,
	"旅行摄影": true,
	"时尚穿搭": true,
	"萌宠生活": true,
	"游戏电竞": true,
	"影视综艺": true,
	"二次元":  true,
	"运动健身": true,
	"职场成长": true,
	"其他内容": true,
}

func ContentSafe(ctx context.Context, post *Post) (int, error) {
	postJson, err := json.Marshal(post)
	if err != nil {
		log.Error("AI大模型请求数据序列化失败", "err", err)
		return ContentSafeWarn, err
	}

	resp, err := fetchModel(ctx, contentSafePrompt, string(postJson), 0.1)

	if err != nil {
		log.Error("AI大模型请求失败", "err", err)
		return ContentSafeWarn, err
	}

	log.Info("AI大模型请求成功", "resp", resp)

	if len(resp.Choices) > 0 {
		if resp.Choices[0].StopReason == ContentSensitive {
			return ContentSafeDeny, nil
		}

		safe := mapContentSafe[resp.Choices[0].Content]
		// AI 没有成功返回，默认为警告，进入人工审核
		if safe == 0 {
			return ContentSafeWarn, nil
		}
		return safe, nil
	}

	return ContentSafeWarn, nil
}

func ImageSafe(ctx context.Context, urls []string) (int, error) {
	if len(urls) == 0 {
		return ContentSafePass, nil
	}
	if len(urls) > 9 {
		return ContentSafeWarn, nil
	}
	resp, err := fetchModelByPicUrls(ctx, imageSafePrompt, urls, 0.1)
	if err != nil {
		log.Error("AI大模型请求失败", "err", err)
		return ContentSafeWarn, err
	}
	if len(resp.Choices) > 0 {
		safe := mapContentSafe[resp.Choices[0].Content]
		if safe == 0 {
			return ContentSafeWarn, nil
		}
		return safe, nil
	}
	return ContentSafeWarn, nil
}

func ContentClassify(ctx context.Context, post *Post) (*ClassifyMessage, error) {
	postJson, err := json.Marshal(post)
	if err != nil {
		log.Error("AI大模型请求数据序列化失败", "err", err)
		return nil, err
	}

	resp, err := fetchModel(ctx, contentClassifyPrompt, string(postJson), 0.1)

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

		// 校验 MainTag 是否存在，不存在直接置空，等待定时任务重新获取
		if !setMainTag[classifyMessage.MainTag] {
			classifyMessage.MainTag = ""
		}

		return classifyMessage, nil
	}

	return nil, nil
}
