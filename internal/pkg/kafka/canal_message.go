package kafka

import (
	"strconv"
	"time"
)

// CanalMessage 定义了 Canal 推送到 Kafka 的 JSON 数据结构
type CanalMessage struct {
	ID       int64    `json:"id"`
	Database string   `json:"database"`
	Table    string   `json:"table"`
	PKNames  []string `json:"pkNames"`
	IsDDL    bool     `json:"isDdl"`
	Type     string   `json:"type"`
	ES       int64    `json:"es"`
	TS       int64    `json:"ts"`
	SQL      string   `json:"sql"`

	// Data 存储变更后的数据
	Data []map[string]interface{} `json:"data"`

	// Old 存储变更前的数据
	Old []map[string]interface{} `json:"old"`

	// 字段类型元数据
	SqlType   map[string]int    `json:"sqlType"`   // JDBC 类型 ID
	MysqlType map[string]string `json:"mysqlType"` // MySQL 类型描述
}

const (
	INSERT = "INSERT"
	DELETE = "DELETE"
)

// StrToString 处理字符串转换string
func StrToString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// StrToUint64 处理字符串转换uint64
func StrToUint64(v interface{}) uint64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case string:
		u, _ := strconv.ParseUint(val, 10, 64)
		return u
	case float64:
		return uint64(val)
	default:
		return 0
	}
}

// StrToInt 处理字符串转换int
func StrToInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case string:
		i, _ := strconv.Atoi(val)
		return i
	case float64:
		return int(val)
	default:
		return 0
	}
}

// StrToDate 处理 "2000-01-01" 这种 date 类型
func StrToDate(v interface{}) time.Time {
	s := StrToString(v)
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02", s)
	return t
}

// StrToDateTime 处理 "2025-12-27 19:44:39" 这种 datetime 类型
func StrToDateTime(v interface{}) time.Time {
	s := StrToString(v)
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02 15:04:05", s)
	return t
}
