package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogEvent is a catalogued log entry: a stable code paired with a static message.
//
// Defining events centrally (see package logger/catalog) instead of inlining
// string literals means:
//   - every log line is searchable by a unique code in any aggregator (Datadog, Loki, …)
//   - message text lives in one place and is never duplicated
//   - callers only supply context fields — not the message or code
type LogEvent struct {
	// Code is the unique, machine-readable event identifier.
	// Convention: SCREAMING_SNAKE_CASE prefixed by domain, e.g. "CALL_CREATE_FAILED".
	Code string

	// Message is the human-readable description logged alongside the code.
	Message string
}

// entry builds the full zap field list: event metadata first, then caller fields.
func (e LogEvent) entry(fields []zap.Field) []zap.Field {
	base := []zap.Field{
		zap.String("event_code", e.Code),
		zap.Time("event_ts", time.Now().UTC()),
	}
	return append(base, fields...)
}

// log dispatches at the requested level, avoiding a switch in every helper.
func (e LogEvent) log(logger *zap.Logger, level zapcore.Level, fields []zap.Field) {
	if ce := logger.Check(level, e.Message); ce != nil {
		ce.Write(e.entry(fields)...)
	}
}

// Debug logs the event at DEBUG level.
func (e LogEvent) Debug(logger *zap.Logger, fields ...zap.Field) {
	e.log(logger, zapcore.DebugLevel, fields)
}

// Info logs the event at INFO level.
func (e LogEvent) Info(logger *zap.Logger, fields ...zap.Field) {
	e.log(logger, zapcore.InfoLevel, fields)
}

// Warn logs the event at WARN level.
func (e LogEvent) Warn(logger *zap.Logger, fields ...zap.Field) {
	e.log(logger, zapcore.WarnLevel, fields)
}

// Error logs the event at ERROR level.
func (e LogEvent) Error(logger *zap.Logger, fields ...zap.Field) {
	e.log(logger, zapcore.ErrorLevel, fields)
}

// Fatal logs the event at FATAL level then calls os.Exit(1).
func (e LogEvent) Fatal(logger *zap.Logger, fields ...zap.Field) {
	logger.Fatal(e.Message, e.entry(fields)...)
}
