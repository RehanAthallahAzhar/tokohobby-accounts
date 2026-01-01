package redisclient

import (
	"context"
	"fmt"
	"log"
	"os"
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
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	return &RedisClient{Client: rdb, log: log}, nil
}

func (rc *RedisClient) Close() {
	if rc.Client != nil {
		err := rc.Client.Close()
		if err != nil {
			log.Printf("Gagal menutup koneksi Redis: %v", err)
		}
	}
}

// Wrapper methods for Redis operations

func (rc *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rc.Client.Set(ctx, key, value, expiration).Err()
}

func (rc *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return rc.Client.Get(ctx, key).Result()
}

func (rc *RedisClient) Del(ctx context.Context, key string) error {
	return rc.Client.Del(ctx, key).Err()
}
