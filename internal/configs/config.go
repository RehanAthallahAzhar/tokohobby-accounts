package configs

import (
	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type AppConfig struct {
	Database  DatabaseConfig
	Migration MigrationConfig
	Redis     RedisConfig
	GRPC      GrpcConfig
	Server    ServerConfig
	RabbitMQ  struct {
		URL string `env:"RABBITMQ_URL,required"`
	}
	Logrus LogrusConfig
}

func LoadConfig(log *logrus.Logger) (*AppConfig, error) {
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
