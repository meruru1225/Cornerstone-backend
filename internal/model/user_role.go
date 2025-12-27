package model

type UserRole struct {
	UserID uint64 `gorm:"primaryKey" json:"user_id"`
	RoleID uint64 `gorm:"primaryKey;index:idx_role_id" json:"role_id"`
}

func (UserRole) TableName() string {
	return "user_roles"
}
