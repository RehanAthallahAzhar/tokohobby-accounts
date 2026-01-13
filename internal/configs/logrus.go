package configs

type LogrusConfig struct {
	Level string `env:"LOG_LEVEL,required"`
}
