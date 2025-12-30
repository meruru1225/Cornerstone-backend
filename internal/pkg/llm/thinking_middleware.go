package llm

import (
	"bytes"
	"io"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/tmc/langchaingo/llms/openai"
)

// ThinkingMiddleware 拦截并修改请求 Body
type ThinkingMiddleware struct {
	Base http.RoundTripper
}

func (m *ThinkingMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	// 读取原始 Body
	body, _ := io.ReadAll(req.Body)
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	// 注入参数：关闭GLM思考
	data["thinking"] = map[string]interface{}{
		"type": "disabled",
	}

	// 写回 Body
	newBody, _ := json.Marshal(data)
	req.Body = io.NopCloser(bytes.NewBuffer(newBody))
	req.ContentLength = int64(len(newBody))

	return m.Base.RoundTrip(req)
}

// NewGLMClient 创建支持思考控制的客户端
func NewGLMClient(apiKey string, baseURL string) (*openai.LLM, error) {
	return openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL(baseURL),
		// 注入自定义 HTTP Client
		openai.WithHTTPClient(&http.Client{
			Transport: &ThinkingMiddleware{Base: http.DefaultTransport},
		}),
	)
}
