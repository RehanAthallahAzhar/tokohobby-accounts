package token

import (
	"context"
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/google/uuid"
)

type TokenService interface {
	GenerateToken(ctx context.Context, user *entities.User) (string, error)
	ValidateToken(ctx context.Context, tokenString string) (isValid bool, userID uuid.UUID, username string, role string, errorMessage string, err error)
	BlacklistToken(ctx context.Context, jti string, expiration time.Duration) error
}
