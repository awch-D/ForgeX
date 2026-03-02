// Package logger provides structured logging for ForgeX using Zap.
package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log  *zap.SugaredLogger
	once sync.Once
)

// Init initializes the global logger with the specified level.
func Init(level string, dev bool) {
	once.Do(func() {
		var cfg zap.Config
		if dev {
			cfg = zap.NewDevelopmentConfig()
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			cfg = zap.NewProductionConfig()
		}

		// Parse level
		var zapLevel zapcore.Level
		if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
			zapLevel = zapcore.InfoLevel
		}
		cfg.Level.SetLevel(zapLevel)

		logger, err := cfg.Build()
		if err != nil {
			panic("failed to initialize logger: " + err.Error())
		}
		log = logger.Sugar()
	})
}

// L returns the global SugaredLogger instance.
func L() *zap.SugaredLogger {
	if log == nil {
		Init("info", true)
	}
	return log
}

// Sync flushes any buffered log entries.
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}
