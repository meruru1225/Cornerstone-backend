package security

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken 生成一个新的 JWT Token
func GenerateToken(userID uint64, roles []string) (string, error) {
	expirationTime := time.Now().Add(JWTExpirationTime)

	claims := &UserClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "Raqtpie",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		return "", fmt.Errorf("签名 Token 失败: %w", err)
	}

	return tokenString, nil
}

// ValidateToken 验证 Token 字符串并解析出 Claims
func ValidateToken(tokenString string) (*UserClaims, error) {
	claims := &UserClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return []byte(JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token 解析失败: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token 无效或已过期")
	}

	return claims, nil
}

// ExtractSignature 从 Token 字符串中提取签名
func ExtractSignature(tokenString string) (string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", errors.New("token 格式不正确")
	}
	return parts[2], nil
}
