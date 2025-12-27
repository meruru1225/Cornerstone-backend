package dto

import "time"

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
