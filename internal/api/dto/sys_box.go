package dto

// SysBoxDTO 系统通知返回对象
type SysBoxDTO struct {
	ID         string         `json:"id"`
	SenderID   uint64         `json:"sender_id"`
	SenderName string         `json:"sender_name"`
	AvatarURL  string         `json:"avatar_url"`
	Type       int8           `json:"type"`      // 1-点赞, 2-收藏, 3-评论, 4-评论点赞, 5-关注
	TargetID   uint64         `json:"target_id"` // 关联的帖子ID
	Content    string         `json:"content"`   // 预览内容
	Payload    map[string]any `json:"payload"`   // 扩展字段
	IsRead     bool           `json:"is_read"`
	CreatedAt  string         `json:"created_at"`
}

// SysBoxUnreadDTO 未读数返回
type SysBoxUnreadDTO struct {
	UnreadCount int64 `json:"unread_count"`
}
