package dto

type ForgetPasswordDTO struct {
	Phone       *string `json:"phone" binding:"required" validate:"min=11,max=11"`
	SmsCode     *string `json:"sms_code" binding:"required" validate:"min=6,max=6"`
	NewPassword *string `json:"new_password" binding:"required" validate:"min=6,max=20"`
}
