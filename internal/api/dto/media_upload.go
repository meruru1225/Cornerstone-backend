package dto

type MediaTempMetadata struct {
	MimeType  string  `json:"mime_type"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Duration  float64 `json:"duration"`
	CreatedAt int64   `json:"created_at"`
}
