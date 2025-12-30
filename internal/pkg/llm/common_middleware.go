package llm

import (
	"Cornerstone/internal/api/config"
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/goccy/go-json"
	"github.com/tmc/langchaingo/llms/openai"
)

// CommonMiddleware 通用中间件：根据 API 路径自动补全厂商私有参数
type CommonMiddleware struct {
	Base http.RoundTripper
}

func (m *CommonMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil || len(body) == 0 {
		return m.Base.RoundTrip(req)
	}

	var data map[string]interface{}
	if err = json.Unmarshal(body, &data); err != nil {
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		return m.Base.RoundTrip(req)
	}

	path := req.URL.Path
	modified := false

	cfg := config.Cfg.LLM

	if strings.Contains(path, "chat/completions") {
		data["thinking"] = map[string]interface{}{
			"type": cfg.ThinkingMode,
		}
		modified = true
	} else if strings.Contains(path, "embeddings") {
		data["dimensions"] = cfg.Dimensions
		data["model"] = cfg.EmbeddingModel
		modified = true
	}

	if modified {
		newBody, _ := json.Marshal(data)
		req.Body = io.NopCloser(bytes.NewBuffer(newBody))
		req.ContentLength = int64(len(newBody))
	} else {
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	return m.Base.RoundTrip(req)
}

// NewGLMClient 创建支持思考控制的客户端
func NewGLMClient(apiKey string, baseURL string) (*openai.LLM, error) {
	return openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL(baseURL),
		// 注入自定义 HTTP Client
		openai.WithHTTPClient(&http.Client{
			Transport: &CommonMiddleware{Base: http.DefaultTransport},
		}),
	)
}
