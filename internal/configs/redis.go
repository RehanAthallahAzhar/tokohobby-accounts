package configs

type RedisConfig struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost"`
	Port     string `env:"REDIS_PORT" envDefault:"6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB,required"`
}
