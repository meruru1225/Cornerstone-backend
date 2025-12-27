package model

type UserDetail struct {
	UserID    uint64  `gorm:"primaryKey"`
	Nickname  string  `gorm:"type:varchar(50);not null"`
	AvatarURL string  `gorm:"type:varchar(512);column:avatar_url;default:'default_avatar.png'"`
	Bio       *string `gorm:"type:varchar(255);default:''"`
	Gender    *uint8  `gorm:"type:tinyint;default:0"`
	Region    *string `gorm:"type:varchar(255)"`
	Birthday  *string `gorm:"type:date"`
}

func (UserDetail) TableName() string {
	return "user_detail"
}
