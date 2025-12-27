package model

type PostTag struct {
	PostID uint64 `gorm:"primaryKey" json:"postId"`
	TagID  uint64 `gorm:"primaryKey;index:idx_tag_id" json:"tagId"`
}

func (PostTag) TableName() string {
	return "post_tags"
}
