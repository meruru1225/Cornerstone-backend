package llm

import (
	"context"
	"errors"
	"fmt"
	log "log/slog"
	"strings"

	"github.com/goccy/go-json"
)

func ImageProcess(ctx context.Context, urls []string, auditOnly bool) (*ContentResponse, error) {
	if len(urls) == 0 {
		return GetPassResponse(), nil
	}
	if len(urls) > 9 {
		return GetWarnResponse(), nil
	}

	prompt := imageProcessPrompt
	if auditOnly {
		prompt = imageAuditOnlyPrompt
	}

	resp, err := fetchModelByPicUrls(ctx, prompt, urls, 0.1)
	if err != nil {
		log.ErrorContext(ctx, "图像处理-AI大模型请求失败", "err", err)
		return nil, err
	}

	log.InfoContext(ctx, "图像处理-AI大模型请求成功", "resp", resp)

	if len(resp.Choices) > 0 {
		if resp.Choices[0].StopReason == ContentSensitive {
			return GetDenyResponse(), nil
		}

		contentResp, err := GetContentResponse(resp.Choices[0].Content)
		if err != nil {
			log.ErrorContext(ctx, "图像处理-AI大模型返回数据解析失败", "err", err, "resp", resp.Choices[0].Content)
			return nil, err
		}
		return contentResp, nil
	}
	return nil, errors.New("图像处理-AI大模型返回数据为空")
}

func ContentProcess(ctx context.Context, content *Content, auditOnly bool) (*ContentResponse, error) {
	prompt := contentProcessPrompt
	if auditOnly {
		prompt = contentAuditOnlyPrompt
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		log.ErrorContext(ctx, "内容处理-AI大模型请求数据序列化失败", "err", err)
		return nil, err
	}

	resp, err := fetchModel(ctx, prompt, string(contentJSON), 0.1)
	if err != nil {
		log.ErrorContext(ctx, "内容处理-AI大模型请求失败", "err", err)
		return nil, err
	}

	log.InfoContext(ctx, "内容处理-AI大模型请求成功", "resp", resp)

	if len(resp.Choices) > 0 {
		if resp.Choices[0].StopReason == ContentSensitive {
			return GetDenyResponse(), nil
		}

		contentResp, err := GetContentResponse(resp.Choices[0].Content)
		if err != nil {
			log.ErrorContext(ctx, "内容处理-AI大模型返回数据解析失败", "err", err, "resp", resp.Choices[0].Content)
			return nil, err
		}
		return contentResp, nil
	}

	return nil, errors.New("内容处理-AI大模型返回数据为空")
}

func Aggressive(ctx context.Context, payload *TagAggressive) (*ContentResponse, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.ErrorContext(ctx, "内容聚合-AI大模型请求数据序列化失败", "err", err)
		return nil, err
	}

	resp, err := fetchModel(ctx, aggressiveTagPrompt, string(payloadJson), 0.1)
	if err != nil {
		log.ErrorContext(ctx, "内容聚合-AI大模型请求失败", "err", err)
		return nil, err
	}

	if len(resp.Choices) > 0 {
		if resp.Choices[0].StopReason == ContentSensitive {
			return GetDenyResponse(), nil
		}

		contentResp, err := GetContentResponse(resp.Choices[0].Content)
		if err != nil {
			log.ErrorContext(ctx, "标签聚合-AI大模型返回数据解析失败", "err", err, "resp", resp.Choices[0].Content)
			return nil, err
		}
		log.InfoContext(ctx, "标签聚合-AI大模型请求成功", "resp", resp)
		return contentResp, nil
	}

	return nil, errors.New("标签聚合-AI大模型返回数据为空")
}

func GetVector(ctx context.Context, content *Content, tags []string, summary string) ([]float32, error) {
	if content == nil {
		return nil, errors.New("内容处理-AI大模型返回数据为空")
	}
	s := fmt.Sprintf("Title: %s\nContent: %s\nTags: %s\nSummary: %s\n", *content.Title, content.Content, strings.Join(tags, ","), summary)
	vector, err := fetchModelEmbedding(ctx, s)
	if err != nil {
		log.ErrorContext(ctx, "内容处理-AI大模型向量获取失败", "err", err)
		return nil, err
	}
	log.InfoContext(ctx, "内容处理-AI大模型向量获取成功", "vector", vector)
	return vector, nil
}

func GetVectorByString(ctx context.Context, s string) ([]float32, error) {
	if s == "" {
		return nil, errors.New("内容处理-AI大模型返回数据为空")
	}
	vector, err := fetchModelEmbedding(ctx, s)
	if err != nil {
		log.ErrorContext(ctx, "内容处理-AI大模型向量获取失败", "err", err)
		return nil, err
	}
	log.InfoContext(ctx, "内容处理-AI大模型向量获取成功", "vector", vector)
	return vector, nil
}
