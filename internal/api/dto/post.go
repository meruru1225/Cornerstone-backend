package dto

// PostDTO 帖子
type PostDTO struct {
	// Post
	ID        uint64 `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// PostMedia
	Medias []*MediasBaseDTO `json:"medias"`

	// User
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// PostBaseDTO 帖子 - 新增或修改
type PostBaseDTO struct {
	ID      uint64           `json:"id"`
	Title   string           `json:"title" binding:"required" validate:"min=1,max=255"`
	Content string           `json:"content" binding:"required" validate:"min=1,max=1000"`
	Medias  []*MediasBaseDTO `json:"Medias" validate:"max=9"`
}

// MediasBaseDTO 媒体 - 基础
type MediasBaseDTO struct {
	MimeType string  `json:"mime_type" binding:"required" validate:"min=1,max=64"`
	MediaURL string  `json:"url" binding:"required" validate:"min=1,max=512"`
	Width    int     `json:"width" binding:"required" validate:"min=1"`
	Height   int     `json:"height" binding:"required" validate:"min=1"`
	Duration int     `json:"duration"`
	CoverURL *string `json:"cover_url,omitempty"`
}
