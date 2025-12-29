package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/redisclient"
	"github.com/go-redis/redis/v8"
)

type SessionRepository interface {
	StoreUserSession(ctx context.Context, token string, user *entities.User, duration time.Duration) error
	GetUserSession(ctx context.Context, token string) (*entities.User, error)
	DeleteUserSession(ctx context.Context, token string) error
}

type sessionRepository struct {
	redisClient *redisclient.RedisClient
}

func NewSessionRepository(redisClient *redisclient.RedisClient) SessionRepository {
	return &sessionRepository{redisClient: redisClient}
}

func (r *sessionRepository) StoreUserSession(ctx context.Context, token string, user *entities.User, duration time.Duration) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user to JSON: %w", err)
	}

	return r.redisClient.Client.SetEX(ctx, token, userJSON, duration).Err()
}

func (r *sessionRepository) GetUserSession(ctx context.Context, token string) (*entities.User, error) {
	val, err := r.redisClient.Client.Get(ctx, token).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found for token: %s", token)
		}
		return nil, fmt.Errorf("failed to retrieve session from Redis: %w", err)
	}

	var user entities.User
	err = json.Unmarshal([]byte(val), &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user from JSON: %w", err)
	}
	return &user, nil
}

func (r *sessionRepository) DeleteUserSession(ctx context.Context, token string) error {
	return r.redisClient.Client.Del(ctx, token).Err()
}
