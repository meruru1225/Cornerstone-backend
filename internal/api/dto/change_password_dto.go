package dto

type ChangePasswordDTO struct {
	OldPassword *string `json:"old_password" binding:"required" validate:"min=6,max=20"`
	NewPassword *string `json:"new_password" binding:"required" validate:"min=6,max=20"`
}
