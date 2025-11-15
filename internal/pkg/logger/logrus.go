package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)

	if os.Getenv("ENV") == "production" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		if os.Getenv("ENV") == "production" {
			level = logrus.InfoLevel
		} else {
			level = logrus.DebugLevel
		}
	}

	log.SetLevel(level)

	return log
}
