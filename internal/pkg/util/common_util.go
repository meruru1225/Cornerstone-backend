package util

import (
	"regexp"
	"strings"
)

var tagRegex = regexp.MustCompile(`#(\S+)`)

// ExtractTags 只负责提取去重后的标签列表
func ExtractTags(rawContent string) []string {
	matches := tagRegex.FindAllStringSubmatch(rawContent, -1)

	tagSet := make(map[string]struct{})
	var tags []string

	for _, m := range matches {
		if len(m) > 1 {
			tagName := m[1]

			tagName = strings.Trim(tagName, ".,，。!?！？")

			if tagName != "" {
				if _, exists := tagSet[tagName]; !exists {
					tagSet[tagName] = struct{}{}
					tags = append(tags, tagName)
				}
			}
		}
	}

	return tags
}

// PtrInt 用于将 int 转换为 *int
func PtrInt(i int) *int {
	return &i
}

// PtrInt64 用于将 int64 转换为 *int64
func PtrInt64(i int64) *int64 {
	return &i
}
