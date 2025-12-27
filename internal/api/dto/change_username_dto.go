package dto

type ChangeUsernameDTO struct {
	Username *string `json:"username" binding:"required" validate:"min=3,max=20"`
}
