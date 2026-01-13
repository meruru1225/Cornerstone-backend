package model

import (
	"time"
)

type User struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Username  *string   `gorm:"type:varchar(50);uniqueIndex:idx_username" json:"username"`
	Phone     *string   `gorm:"type:varchar(30);uniqueIndex:idx_phone" json:"phone"`
	Password  *string   `gorm:"type:varchar(255)" json:"password"`
	IsBan     bool      `gorm:"type:tinyint(1);default:0" json:"is_ban"`
	IsDelete  bool      `gorm:"type:tinyint(1);default:0" json:"is_delete"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserDetail UserDetail `gorm:"foreignKey:UserID;references:ID" json:"user_detail"`
	UserRoles  []UserRole `gorm:"foreignKey:UserID;references:ID" json:"user_roles"`
}

func (User) TableName() string {
	return "users"
}
