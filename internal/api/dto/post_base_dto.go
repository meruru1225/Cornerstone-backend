package dto

type PostBaseDTO struct {
	ID           uint64   `json:"id"`
	Title        string   `json:"title" binding:"required" validate:"min=1,max=255"`
	Content      string   `json:"content" binding:"required" validate:"min=1,max=1000"`
	Medias       []Medias `json:"Medias" validate:"max=9"`
	MediaChanged bool     `json:"mediaChanged"`
}

type Medias struct {
	MediaName string  `json:"mediaName" binding:"required" validate:"min=1,max=255"`
	MimeType  string  `json:"mimeType" binding:"required" validate:"min=1,max=255"`
	Width     int     `json:"width" binding:"required" validate:"min=1"`
	Height    int     `json:"height" binding:"required" validate:"min=1"`
	CoverURL  *string `json:"coverUrl,omitempty"`
}
