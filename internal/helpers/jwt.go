package helpers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTHelper struct {
	secretKey []byte
}

func NewJWTHelper(secret string) *JWTHelper {
	return &JWTHelper{
		secretKey: []byte(secret),
	}
}

func (h *JWTHelper) GenerateToken(username string) (string, error) {
	expirationTime := time.Now().Add(60 * time.Minute)

	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(h.secretKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}
