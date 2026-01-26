package llm

import "github.com/tmc/langchaingo/llms"

// DefineGeneralSearchTool 定义搜索工具的元数据
func DefineGeneralSearchTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "search_community_posts",
			Description: "搜索社区内的各类笔记、经验分享、生活指南、攻略及讨论内容。当你需要获取站内真实用户的见解、实地考察信息或专业领域知识时，请调用此工具。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "搜索关键词，例如：'上海咖啡店推荐'、'新手如何入门自媒体'、'2026年手机选购建议'",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// DefineGetPostURLTool 定义获取站内帖子链接的工具
func DefineGetPostURLTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "get_post_url",
			Description: "获取指定站内帖子 ID 的官方访问链接。当你从 search_community_posts 中发现高质量内容并决定向用户推荐具体跳转地址时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"post_id": map[string]any{
						"type":        "integer",
						"description": "帖子的唯一 ID (从搜索结果中获得)",
					},
				},
				"required": []string{"post_id"},
			},
		},
	}
}

// DefineWebSearchTool 定义 DuckDuckGo 搜索工具
func DefineWebSearchTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "web_search",
			Description: "使用 DuckDuckGo 搜索互联网上的实时信息、新闻或技术文档。当你需要站外广泛的背景知识时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "搜索关键词",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// DefineWebFetchTool 定义网页内容抓取工具
func DefineWebFetchTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "web_fetch",
			Description: "抓取并阅读指定 URL 的网页详细正文内容。当搜索摘要信息不足以回答问题时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "完整的网页 URL",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}
