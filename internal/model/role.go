package model

type Role struct {
	ID          uint64  `gorm:"primaryKey"`
	Name        string  `gorm:"type:varchar(50);uniqueIndex:idx_role_name;not null"`
	Description *string `gorm:"type:varchar(255)"`
}

func (Role) TableName() string {
	return "roles"
}
