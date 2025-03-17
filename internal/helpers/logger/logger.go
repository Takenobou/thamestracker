package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

func InitLogger() {
	var zapLogger *zap.Logger
	var err error

	if os.Getenv("APP_ENV") == "dev" {
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.ConsoleSeparator = "  "
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.MessageKey = "msg"
		zapLogger, err = cfg.Build()
	} else {
		zapLogger, err = zap.NewProduction()
	}
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	Logger = zapLogger.Sugar()
}
