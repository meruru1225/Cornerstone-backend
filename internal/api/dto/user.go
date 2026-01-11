package dto

import "time"

// UserDTO 用户
type UserDTO struct {
	UserID    *uint64    `json:"user_id,omitempty"`
	Phone     *string    `json:"phone,omitempty"`
	Nickname  *string    `json:"nickname,omitempty"`
	AvatarURL *string    `json:"avatar_url,omitempty"`
	Bio       *string    `json:"bio,omitempty" validate:"omitempty,max=200"`
	Gender    *uint8     `json:"gender,omitempty" validate:"omitempty,min=0,max=1"`
	Region    *string    `json:"region,omitempty"`
	Birthday  *string    `json:"birthday,omitempty" validate:"omitempty,datetime=2006-01-02"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// GetUserByConditionDTO 搜索用户
type GetUserByConditionDTO struct {
	ID       *uint64 `json:"id,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Username *string `json:"username,omitempty"`
	Nickname *string `json:"nickname,omitempty"`
}

// RegisterDTO 注册
type RegisterDTO struct {
	// 方式一 使用 用户名&密码
	Username *string `json:"username"`
	Password *string `json:"password"`

	// 方式二 使用 手机号&临时签发令牌
	Phone      *string `json:"phone"`
	PhoneToken *string `json:"phone_token"`

	Nickname string  `json:"nickname" validate:"required,min=1,max=15"`
	Bio      *string `json:"bio"`
	Gender   uint8   `json:"gender"`
	Region   *string `json:"region"`
	Birthday string  `json:"birthday" validate:"required"`
}

// ForgetPasswordDTO 忘记密码
type ForgetPasswordDTO struct {
	Phone       *string `json:"phone" binding:"required" validate:"min=11,max=11"`
	SmsCode     *string `json:"sms_code" binding:"required" validate:"min=6,max=6"`
	NewPassword *string `json:"new_password" binding:"required" validate:"min=6,max=20"`
}

// CredentialDTO 登录凭证
type CredentialDTO struct {
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Phone    *string `json:"phone,omitempty"`
}

// ChangeUsernameDTO 修改用户名
type ChangeUsernameDTO struct {
	Username *string `json:"username" binding:"required" validate:"min=3,max=20"`
}

// ChangePhoneDTO 修改手机号
type ChangePhoneDTO struct {
	Token    *string `json:"token" binding:"required" validate:"required"`
	NewPhone *string `json:"new_phone" binding:"required" validate:"required,min=11,max=11"`
}

// ChangePasswordDTO 修改密码
type ChangePasswordDTO struct {
	OldPassword *string `json:"old_password" binding:"required" validate:"min=6,max=20"`
	NewPassword *string `json:"new_password" binding:"required" validate:"min=6,max=20"`
}
