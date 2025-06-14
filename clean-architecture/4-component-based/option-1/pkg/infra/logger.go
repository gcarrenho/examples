package infra

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func InitLogrusLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

func InitZapLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}
