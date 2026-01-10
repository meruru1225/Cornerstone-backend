package dto

// UserContentTrendDTO 创作者全维度趋势数据
type UserContentTrendDTO struct {
	UserID   uint64           `json:"user_id"`
	Days     int              `json:"days"`
	Likes    []*PostMetricDTO `json:"likes"`
	Collects []*PostMetricDTO `json:"collects"`
	Comments []*PostMetricDTO `json:"comments"`
	Views    []*PostMetricDTO `json:"views"`
}
