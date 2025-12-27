package dto

type ChangePhoneDTO struct {
	Token    *string `json:"token" binding:"required" validate:"required"`
	NewPhone *string `json:"new_phone" binding:"required" validate:"required,min=11,max=11"`
}
