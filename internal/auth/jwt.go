package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID         uint64 `json:"user_id"`
	Role           string `json:"role"`
	ImpersonatedBy uint64 `json:"impersonated_by,omitempty"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint64, role, secret, expiry string) (string, error) {
	return GenerateTokenAs(userID, role, 0, secret, expiry)
}

// GenerateTokenAs mints a token for userID/role. When impersonatedBy > 0
// the claim is embedded so audit logs can trace who initiated the session.
func GenerateTokenAs(userID uint64, role string, impersonatedBy uint64, secret, expiry string) (string, error) {
	dur, err := time.ParseDuration(expiry)
	if err != nil {
		return "", fmt.Errorf("parse expiry: %w", err)
	}
	claims := Claims{
		UserID:         userID,
		Role:           role,
		ImpersonatedBy: impersonatedBy,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(dur)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
