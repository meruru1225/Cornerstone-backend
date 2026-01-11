package model

import (
	"time"
)

type User struct {
	ID        uint64  `gorm:"primaryKey"`
	Username  *string `gorm:"type:varchar(50);uniqueIndex:idx_username"`
	Phone     *string `gorm:"type:varchar(30);uniqueIndex:idx_phone"`
	Password  *string `gorm:"type:varchar(255)"`
	IsBan     bool    `gorm:"type:tinyint(1);default:0"`
	IsDelete  bool    `gorm:"type:tinyint(1);default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserDetail UserDetail `gorm:"foreignKey:UserID;references:ID"`
	UserRoles  []UserRole `gorm:"foreignKey:UserID;references:ID"`
}

func (User) TableName() string {
	return "users"
}
