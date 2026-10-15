package configs

import (
	"log/slog"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type AppConfig struct {
	Database  DatabaseConfig
	Migration MigrationConfig
	Redis     RedisConfig
	GRPC      GrpcConfig
	Server    ServerConfig
	RabbitMQ  RabbitMQConfig
	Kafka     KafkaConfig
	Tracer    TracerConfig
}

func LoadConfig(log *slog.Logger) (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		log.Warn("Warn: Failed to load .env.")
	}

	cfg := &AppConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	log.Info("Configuration loaded successfully.")
	return cfg, nil
}
