package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

var logger *zap.Logger

func init() {
	logger, _ = newConfig().Build()
}

func newConfig() zap.Config {
	return zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			TimeKey:      "time",
			NameKey:      "name",
			CallerKey:    "caller",
			LineEnding:   "\n",
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeTime:   zapcore.TimeEncoderOfLayout(time.RFC3339),
			EncodeCaller: zapcore.FullCallerEncoder,
		},
		OutputPaths: []string{"stderr"},
	}
}

func GetLogger() *zap.Logger {
	return logger
}
