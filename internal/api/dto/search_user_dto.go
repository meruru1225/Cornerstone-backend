package dto

type SearchUserDTO struct {
	ID       *uint64 `json:"id,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Username *string `json:"username,omitempty"`
	Nickname *string `json:"nickname,omitempty"`
}
