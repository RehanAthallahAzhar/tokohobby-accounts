package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/redisclient"
)

type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID string, refreshToken string, ttl time.Duration) error
	ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}

type refreshTokenRepo struct {
	redis *redisclient.RedisClient
}

func NewRefreshTokenRepository(redis *redisclient.RedisClient) RefreshTokenRepository {
	return &refreshTokenRepo{
		redis: redis,
	}
}

func (r *refreshTokenRepo) StoreRefreshToken(ctx context.Context, userID string, refreshToken string, ttl time.Duration) error {
	// Key = Token, Value = UserID
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	return r.redis.Set(ctx, key, userID, ttl)
}

func (r *refreshTokenRepo) ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	userID, err := r.redis.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (r *refreshTokenRepo) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	return r.redis.Del(ctx, key)
}
