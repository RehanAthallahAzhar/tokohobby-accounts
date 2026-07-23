package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/configs"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type RedisClient struct {
	Client *redis.Client
	log    *logrus.Logger
}

func NewRedisClient(cfg *configs.RedisConfig, log *logrus.Logger) (*RedisClient, error) {
	redisAddr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Ping (Fail-Fast)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect/ping redis at %s: %w", redisAddr, err)
	}
	log.Infof("Connected to Redis successfully at %s", redisAddr)
	return &RedisClient{Client: rdb, log: log}, nil
}

func (rc *RedisClient) Close() {
	if rc.Client != nil {
		err := rc.Client.Close()
		if err != nil {
			rc.log.WithError(err).Error("Failed to close redis connection")
			return
		}
		rc.log.Info("Redis connection closed successfully")
	}
}

func (rc *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rc.Client.Set(ctx, key, value, expiration).Err()
}

func (rc *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return rc.Client.Get(ctx, key).Result()
}

func (rc *RedisClient) Del(ctx context.Context, key string) error {
	return rc.Client.Del(ctx, key).Err()
}
