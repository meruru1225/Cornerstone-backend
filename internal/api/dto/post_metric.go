package dto

// PostMetricDTO 帖子指标趋势点
type PostMetricDTO struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

// PostTrendDTO 帖子趋势返回包装
type PostTrendDTO struct {
	PostID   uint64           `json:"post_id"`
	Days     int              `json:"days"` // 7 或 30
	Likes    []*PostMetricDTO `json:"likes"`
	Comments []*PostMetricDTO `json:"comments"`
	Collects []*PostMetricDTO `json:"collects"`
	Views    []*PostMetricDTO `json:"views"`
}
