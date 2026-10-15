package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}
