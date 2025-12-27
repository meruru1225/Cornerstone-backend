package kafka

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
