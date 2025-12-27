package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	JWTSecret         string = "Raqtpie"
	JWTExpirationTime        = time.Hour * 24
)

// UserClaims 定义了我们 Token 中需要包含的业务信息
type UserClaims struct {
	UserID uint64   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}
