package llm

import (
	"strings"

	"github.com/goccy/go-json"
)

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

const (
	ContentSafePass = iota + 1
	ContentSafeWarn
	ContentSafeDeny

	ContentSafePassStr = "1"
	ContentSafeWarnStr = "2"
	ContentSafeDenyStr = "3"
)

var mapContentSafe = map[string]int{
	ContentSafePassStr: ContentSafePass,
	ContentSafeWarnStr: ContentSafeWarn,
	ContentSafeDenyStr: ContentSafeDeny,
}

type ContentResponse struct {
	Status  int
	MainTag string
	Tags    []string
}

type ReturnResponse struct {
	Status  string   `json:"status"`
	MainTag string   `json:"main_tag"`
	Tags    []string `json:"tags"`
}

func GetContentResponse(s string) (*ContentResponse, error) {
	cleaned := strings.TrimSpace(s)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var temp ReturnResponse
	err := json.Unmarshal([]byte(cleaned), &temp)
	if err != nil {
		return nil, err
	}

	res := &ContentResponse{
		Status:  mapContentSafe[temp.Status],
		MainTag: temp.MainTag,
		Tags:    temp.Tags,
	}

	// 校验status
	if res.Status == 0 {
		res.Status = ContentSafeWarn
	}

	// 校验 MainTag 是否存在，不存在直接置空，等待定时任务重新获取
	if !setMainTag[res.MainTag] {
		res.MainTag = ""
	}
	return res, nil
}

func GetPassResponse() *ContentResponse {
	return &ContentResponse{
		Status:  ContentSafePass,
		MainTag: "",
		Tags:    nil,
	}
}

func GetWarnResponse() *ContentResponse {
	return &ContentResponse{
		Status:  ContentSafeWarn,
		MainTag: "",
		Tags:    nil,
	}
}

func GetDenyResponse() *ContentResponse {
	return &ContentResponse{
		Status:  ContentSafeDeny,
		MainTag: "",
		Tags:    nil,
	}
}
