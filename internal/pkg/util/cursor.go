package util

import (
	"encoding/base64"
	"hash/crc32"

	"github.com/goccy/go-json"
)

// EncodeCursor 将 ES 返回的 Sort 值数组编码为 Base64 字符串
func EncodeCursor(sortValues []interface{}) string {
	if len(sortValues) == 0 {
		return ""
	}
	b, _ := json.Marshal(sortValues)
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeCursor 将前端传来的 Base64 字符串解码为 Sort 值数组
func DecodeCursor(cursor string) ([]interface{}, error) {
	if cursor == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	var sortValues []interface{}
	err = json.Unmarshal(b, &sortValues)
	return sortValues, err
}

// HashSessionID 将 UUID 字符串转为 int64 种子
func HashSessionID(sessionID string) int64 {
	if sessionID == "" {
		return 0
	}
	return int64(crc32.ChecksumIEEE([]byte(sessionID)))
}
