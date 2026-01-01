package llm

import (
	"Cornerstone/internal/pkg/es"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// HandleFunction 工具处理器函数签名
type HandleFunction func(context.Context, string) (string, error)

// ToolHandler 工具处理器
type ToolHandler struct {
	postRepo es.PostRepo
}

// NewToolHandler 创建工具处理器实例
func NewToolHandler(postRepo es.PostRepo) *ToolHandler {
	return &ToolHandler{postRepo: postRepo}
}

// GetHandleFunction 返回绑定了当前实例的工具映射表
func (s *ToolHandler) GetHandleFunction(funcName string) HandleFunction {
	return map[string]HandleFunction{
		"search_community_posts": s.SearchCommunityPosts,
	}[funcName]
}

// SearchCommunityPosts 实现了具体的搜索逻辑
func (s *ToolHandler) SearchCommunityPosts(ctx context.Context, argsJson string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJson), &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	vector, err := fetchModelEmbedding(ctx, args.Query)
	if err != nil {
		return "", err
	}

	posts, err := s.postRepo.HybridSearch(ctx, args.Query, vector, 0, 10)
	if err != nil {
		return "", err
	}

	if len(posts) == 0 {
		return "未找到任何相关的站内笔记。", nil
	}

	var builder strings.Builder
	builder.WriteString("以下是为你找到的站内相关笔记：\n\n")

	for i, post := range posts {
		displayContent := post.Content
		if len(displayContent) > 300 {
			displayContent = displayContent[:300] + "..."
		}

		item := fmt.Sprintf("### 笔记 %d\n- **标题**: %s\n- **作者**: %s\n- **内容**: %s\n- **AI总结**: %s\n---\n",
			i+1, post.Title, post.UserNickname, displayContent, post.AISummary)
		builder.WriteString(item)
	}
	return builder.String(), nil
}
