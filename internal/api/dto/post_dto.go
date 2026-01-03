package dto

type PostDTO struct {
	// Post
	ID        uint64 `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// PostMedia
	Medias []PostMediaDTO `json:"medias"`

	// User
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type PostMediaDTO struct {
	ID        uint64  `json:"id"`
	PostID    uint64  `json:"postId"`
	MimeType  string  `json:"mime_type"`
	MediaURL  string  `json:"media_url"`
	SortOrder int8    `json:"sort_order"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Duration  int     `json:"duration"`
	CoverURL  *string `json:"coverUrl,omitempty"`
}
