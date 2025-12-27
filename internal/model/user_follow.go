package model

import "time"

type UserFollow struct {
	FollowerID  uint64    `gorm:"primaryKey" json:"followerId"`
	FollowingID uint64    `gorm:"primaryKey;index:idx_following_id" json:"followingId"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (UserFollow) TableName() string {
	return "user_follows"
}
