package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type HubClaims struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// IssueHubToken signs a hub_token (HMAC-SHA256 JWT).
// secret = HUB_PRIVATE_KEY hex string
// ttlSeconds = token lifetime in seconds (recommended: 7*24*3600)
func IssueHubToken(uid, email, secret string, ttlSeconds int64) (string, error) {
	claims := HubClaims{
		UID:   uid,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttlSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyHubToken verifies and parses a hub_token.
func VerifyHubToken(tokenStr, secret string) (*HubClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &HubClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*HubClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
