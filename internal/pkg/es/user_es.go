package es

import "time"

// UserES 对应 user_index 的文档结构
type UserES struct {
	UserID         uint64    `json:"user_id"`
	Nickname       string    `json:"nickname"`
	Bio            string    `json:"bio"`
	AvatarURL      string    `json:"avatar_url"`
	Gender         int       `json:"gender"`
	Region         string    `json:"region"`
	Birthday       time.Time `json:"birthday"`
	FollowersCount int       `json:"followers_count"`
	FollowingCount int       `json:"following_count"`
}
