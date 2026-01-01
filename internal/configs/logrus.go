package configs

type LogrusConfig struct {
	Level string `env:"LOGRUS_LEVEL,required"`
}
