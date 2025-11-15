package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/RehanAthallahAzhar/shopeezy-accounts/internal/pkg/redisclient"

	"github.com/go-redis/redis/v8"
)

type JWTBlacklistRepository interface {
	AddToBlacklist(ctx context.Context, jti string, expiration time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

type jwtBlacklistRepository struct {
	redisClient *redisclient.RedisClient
}

func NewJWTBlacklistRepository(redisClient *redisclient.RedisClient) JWTBlacklistRepository {
	return &jwtBlacklistRepository{redisClient: redisClient}
}

func (r *jwtBlacklistRepository) AddToBlacklist(ctx context.Context, jti string, expiration time.Duration) error {
	key := fmt.Sprintf("jwt:blacklist:%s", jti)
	return r.redisClient.Client.Set(ctx, key, "blacklisted", expiration).Err()
}

func (r *jwtBlacklistRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("jwt:blacklist:%s", jti)
	_, err := r.redisClient.Client.Get(ctx, key).Result()
	if err == nil {
		return true, nil
	}
	if err == redis.Nil {
		return false, nil
	}
	return false, fmt.Errorf("failed to check blacklist: %w", err)
}
