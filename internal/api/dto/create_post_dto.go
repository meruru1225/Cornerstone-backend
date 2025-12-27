package dto

type CreatePostDTO struct {
	Title       string   `json:"title" binding:"required" validate:"min=1,max=255"`
	Content     string   `json:"content" binding:"required" validate:"min=1,max=1000"`
	ObjectNames []string `json:"objectNames" validate:"max=9"`
}
