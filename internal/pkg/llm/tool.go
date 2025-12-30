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
