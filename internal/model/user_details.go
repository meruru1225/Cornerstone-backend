package model

type UserDetail struct {
	UserID    uint64  `gorm:"primaryKey" json:"user_id"`
	Nickname  string  `gorm:"type:varchar(50);not null" json:"nickname"`
	AvatarURL string  `gorm:"type:varchar(512);column:avatar_url;default:'default_avatar.png'" json:"avatar_url"`
	Bio       *string `gorm:"type:varchar(255);default:''" json:"bio"`
	Gender    *uint8  `gorm:"type:tinyint;default:0" json:"gender"`
	Region    *string `gorm:"type:varchar(255)" json:"region"`
	Birthday  *string `gorm:"type:date" json:"birthday"`

	FollowersCount int64 `gorm:"type:int;default:0" json:"-"`
	FollowingCount int64 `gorm:"type:int;default:0" json:"-"`
}

func (UserDetail) TableName() string {
	return "user_detail"
}
