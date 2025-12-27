package model

type UserTag struct {
	UserID uint64 `gorm:"primaryKey"`
	TagID  uint64 `gorm:"primaryKey;index:idx_tag_id"`
}

func (UserTag) TableName() string {
	return "user_tags"
}
