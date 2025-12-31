package es

import "time"

// PostES 写入 ES 的完整文档
type PostES struct {
	ID            uint64        `json:"id"`
	UserID        uint64        `json:"user_id"`
	Status        int           `json:"status"`
	Title         string        `json:"title"`
	Content       string        `json:"content"`
	ContentVector []float32     `json:"content_vector,omitempty"`
	MainTag       string        `json:"main_tag"`
	UserTags      []string      `json:"user_tags"`
	AITags        []string      `json:"ai_tags"`
	AISummary     string        `json:"ai_summary"`
	Media         []PostMediaES `json:"media"`
	UserNickname  string        `json:"user_nickname"`
	UserAvatar    string        `json:"user_avatar"`
	LikesCount    int           `json:"likes_count"`
	CommentsCount int           `json:"comments_count"`
	CollectsCount int           `json:"collects_count"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// PostMediaES 对应 Mapping 中的 media 对象
type PostMediaES struct {
	Type     string  `json:"type"`
	URL      string  `json:"url"`
	Cover    *string `json:"cover,omitempty"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Duration int     `json:"duration"`
}
