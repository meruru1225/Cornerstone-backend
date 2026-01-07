package dto

// UserMetricDTO 用户指标趋势点
type UserMetricDTO struct {
	Date  string `json:"date"`  // 格式化后的日期：2026-01-07
	Value int    `json:"value"` // 具体的指标数值（如粉丝数）
}

// UserTrendDTO 用户趋势返回包装
type UserTrendDTO struct {
	UserID uint64           `json:"user_id"`
	Days   int              `json:"days"` // 7 或 30
	List   []*UserMetricDTO `json:"list"`
}
