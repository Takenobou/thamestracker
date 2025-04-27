package logger

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

// RequestLogger returns a Fiber middleware that logs each HTTP request in JSON.
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate or retrieve request ID
		reqID := uuid.New().String()
		c.Set("X-Request-ID", reqID)

		start := time.Now()
		err := c.Next()
		latency := time.Since(start).Milliseconds()

		// Structured log
		Logger.Infow("http_request",
			"module", "api",
			"request_id", reqID,
			"method", c.Method(),
			"path", c.OriginalURL(),
			"status", c.Response().StatusCode(),
			"latency_ms", latency,
			"error", err,
		)
		return err
	}
}
