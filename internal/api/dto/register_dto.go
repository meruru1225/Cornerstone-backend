package dto

type RegisterDTO struct {
	// 方式一 使用 用户名&密码
	Username *string `json:"username"`
	Password *string `json:"password"`

	// 方式二 使用 手机号&临时签发令牌
	Phone      *string `json:"phone"`
	PhoneToken *string `json:"phone_token"`

	Nickname  string  `json:"nickname" validate:"required,min=1,max=15"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	Gender    uint8   `json:"gender,omitempty"`
	Region    *string `json:"region,omitempty"`
	Birthday  string  `json:"birthday,omitempty" validate:"required"`
}
