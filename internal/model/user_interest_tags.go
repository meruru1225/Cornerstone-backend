package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"
)

type UserInterestTags struct {
	UserID    uint64      `gorm:"primaryKey" json:"user_id"`
	Interests InterestMap `gorm:"type:json;not null" json:"interests"` // 存储 TagID:Score 快照
	UpdatedAt time.Time   `json:"updated_at"`
}

func (UserInterestTags) TableName() string {
	return "user_interest_tags"
}

// InterestMap 存储标签得分: map[tag_name]score
type InterestMap map[string]int64

func (i InterestMap) Value() (driver.Value, error) {
	return json.Marshal(i)
}

func (i *InterestMap) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, i)
}
