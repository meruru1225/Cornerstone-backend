package dto

type CredentialDTO struct {
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Phone    *string `json:"phone,omitempty"`
}
