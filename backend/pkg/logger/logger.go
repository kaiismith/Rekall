package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a configured zap.Logger.
//
// In development mode a human-readable console encoder is used.
// In production a JSON encoder is used with caller information.
func New(level, format string, isDevelopment bool) (*zap.Logger, error) {
	zapLevel, err := parseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("logger: invalid log level %q: %w", level, err)
	}

	if isDevelopment {
		cfg := zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return cfg.Build()
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if format == "console" {
		cfg.Encoding = "console"
	}

	return cfg.Build()
}

// WithComponent returns a child logger pre-tagged with the given component name.
// Use this at service or handler construction so every log line emitted by that
// component is automatically tagged without repeating zap.String("component", …)
// at each call site.
//
//	logger = logger.WithComponent(logger, "call_service")
func WithComponent(l *zap.Logger, component string) *zap.Logger {
	return l.With(zap.String("component", component))
}

// parseLevel converts a string level name to a zapcore.Level.
func parseLevel(level string) (zapcore.Level, error) {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zapcore.InfoLevel, err
	}
	return l, nil
}
