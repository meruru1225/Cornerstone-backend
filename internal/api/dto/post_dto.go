package dto

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
