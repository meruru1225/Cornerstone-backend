package dto

type PostBaseDTO struct {
	ID      uint64          `json:"id"`
	Title   string          `json:"title" binding:"required" validate:"min=1,max=255"`
	Content string          `json:"content" binding:"required" validate:"min=1,max=1000"`
	Medias  []MediasBaseDTO `json:"Medias" validate:"max=9"`
}

type MediasBaseDTO struct {
	MimeType string  `json:"mime_type" binding:"required" validate:"min=1,max=64"`
	MediaURL string  `json:"url" binding:"required" validate:"min=1,max=512"`
	Width    int     `json:"width" binding:"required" validate:"min=1"`
	Height   int     `json:"height" binding:"required" validate:"min=1"`
	Duration int     `json:"duration"`
	CoverURL *string `json:"cover_url,omitempty"`
}
