package dto

type PostDTO struct {
	// Post
	ID        uint64 `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	// PostMedia
	Medias []PostMediaDTO `json:"medias"`

	// User
	UserID    uint64 `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

type PostMediaDTO struct {
	ID        uint64  `json:"id"`
	PostID    uint64  `json:"postId"`
	FileType  string  `json:"fileType"`
	MediaURL  string  `json:"mediaUrl"`
	SortOrder int8    `json:"sortOrder"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Duration  int     `json:"duration"`
	CoverURL  *string `json:"coverUrl,omitempty"`
}
