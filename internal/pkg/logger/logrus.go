package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)

	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		if os.Getenv("ENV") == "production" {
			level = logrus.InfoLevel
			log.SetFormatter(&logrus.JSONFormatter{})
		} else {
			level = logrus.DebugLevel
			log.SetFormatter(&logrus.TextFormatter{
				FullTimestamp: true,
				ForceColors:   true,
			})
		}
	}

	log.SetLevel(level)

	return log
}
