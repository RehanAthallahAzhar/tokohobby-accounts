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
}

func LoadConfig(log *logrus.Logger) (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		log.Warn("Peringatan: Gagal memuat file .env.")
	}

	cfg := &AppConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	log.Info("Konfigurasi terstruktur berhasil dimuat.")
	return cfg, nil
}
