package llm

import (
	"context"
	"errors"
	log "log/slog"

	"github.com/goccy/go-json"
)

func ImageProcess(ctx context.Context, urls []string) (*ContentResponse, error) {
	if len(urls) == 0 {
		return GetPassResponse(), nil
	}
	if len(urls) > 9 {
		return GetWarnResponse(), nil
	}

	resp, err := fetchModelByPicUrls(ctx, imageProcessPrompt, urls, 0.1)
	if err != nil {
		log.Error("图像处理-AI大模型请求失败", "err", err)
		return nil, err
	}

	log.Info("图像处理-AI大模型请求成功", "resp", resp)

	if len(resp.Choices) > 0 {
		if resp.Choices[0].StopReason == ContentSensitive {
			return GetDenyResponse(), nil
		}

		contentResp, err := GetContentResponse(resp.Choices[0].Content)
		if err != nil {
			log.Error("图像处理-AI大模型返回数据解析失败", "err", err)
			return nil, err
		}
		return contentResp, nil
	}
	return nil, errors.New("图像处理-AI大模型返回数据为空")
}

func ContentProcess(ctx context.Context, content *Content) (*ContentResponse, error) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		log.Error("内容处理-AI大模型请求数据序列化失败", "err", err)
		return nil, err
	}

	resp, err := fetchModel(ctx, contentProcessPrompt, string(contentJSON), 0.1)
	if err != nil {
		log.Error("内容处理-AI大模型请求失败", "err", err)
		return nil, err
	}

	log.Info("内容处理-AI大模型请求成功", "resp", resp)

	if len(resp.Choices) > 0 {
		contentResp, err := GetContentResponse(resp.Choices[0].Content)
		if err != nil {
			log.Error("内容处理-AI大模型返回数据解析失败", "err", err)
			return nil, err
		}
		return contentResp, nil
	}

	return nil, errors.New("内容处理-AI大模型返回数据为空")
}

func AggressiveTag(ctx context.Context, payload *TagAggressive) (*ContentResponse, error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Error("标签聚合-AI大模型请求数据序列化失败", "err", err)
		return nil, err
	}

	resp, err := fetchModel(ctx, aggressiveTagPrompt, string(payloadJson), 0.1)
	if err != nil {
		log.Error("标签聚合-AI大模型请求失败", "err", err)
		return nil, err
	}

	if len(resp.Choices) > 0 {
		contentResp, err := GetContentResponse(resp.Choices[0].Content)
		if err != nil {
			log.Error("标签聚合-AI大模型返回数据解析失败", "err", err)
			return nil, err
		}
		return contentResp, nil
	}

	return nil, errors.New("标签聚合-AI大模型返回数据为空")
}
