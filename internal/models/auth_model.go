package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

type GenerateAccessTokenRequest struct {
	UserID   uuid.UUID `json:"user_id" validate:"required"`
	Username string    `json:"username" validate:"required"`
	Role     string    `json:"role" validate:"required"`
}

type GenerateRefreshTokenRequest struct {
	UserID   uuid.UUID `json:"user_id" validate:"required"`
	Username string    `json:"username" validate:"required"`
}

type RefreshSessionRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type GenerateTokenResult struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

type ActivityMetadata struct {
	SessionID string
	IPAddress string
	UserAgent string
}
