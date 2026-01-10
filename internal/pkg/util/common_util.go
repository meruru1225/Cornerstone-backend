package util

import (
	"regexp"
	"strconv"
	"strings"
	"time"
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

// PtrFloat32 用于将 float32 转换为 *float32
func PtrFloat32(f float32) *float32 {
	return &f
}

// StrSliceToUInt64Slice 将字符串切片转换为整数切片
func StrSliceToUInt64Slice(strSlice []string) ([]uint64, error) {
	intSlice := make([]uint64, len(strSlice))
	for i, str := range strSlice {
		num, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return nil, err
		}
		intSlice[i] = num
	}
	return intSlice, nil
}

// GetMidnight 获取当天 0 点时间
func GetMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
