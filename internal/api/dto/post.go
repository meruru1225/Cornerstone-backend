package dto

// PostDTO 帖子
type PostDTO struct {
	// Post
	ID        uint64 `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Status    *int8  `json:"status,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// PostMedia
	Medias []*MediasBaseDTO `json:"medias"`

	// User
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// PostWaterfallDTO 帖子瀑布流
type PostWaterfallDTO struct {
	List       []*PostDTO `json:"list"`
	NextCursor string     `json:"next_cursor,omitempty"`
	HasMore    bool       `json:"has_more"`
}

// PostBaseDTO 帖子 - 新增或修改
type PostBaseDTO struct {
	Title        string           `json:"title" binding:"required" validate:"min=1,max=255"`
	Content      string           `json:"content" binding:"required" validate:"min=1,max=20000"`
	PlainContent string           `json:"plain_content" binding:"required" validate:"min=1,max=2000"`
	Medias       []*MediasBaseDTO `json:"medias" validate:"max=9"`
}

// MediasBaseDTO 媒体 - 基础
type MediasBaseDTO struct {
	MimeType string  `json:"mime_type" binding:"required" validate:"min=1,max=64"`
	MediaURL string  `json:"url" binding:"required" validate:"min=1,max=512"`
	Width    int     `json:"width" binding:"required" validate:"min=1"`
	Height   int     `json:"height" binding:"required" validate:"min=1"`
	Duration float64 `json:"duration"`
	CoverURL *string `json:"cover_url,omitempty"`
}

type PostListDTO struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type RecommendPostReq struct {
	Cursor    string `form:"cursor"`
	PageSize  int    `form:"page_size,default=10"`
	SessionID string `form:"session_id"`
}

type PostDeleteDTO struct {
	ID uint64 `json:"id" binding:"required"`
}

type PostUpdateDTO struct {
	Status int `json:"status" binding:"required" validate:"oneof=0 1 2"`
}
