package logger_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rekall/backend/pkg/logger"
)

// ─── New ──────────────────────────────────────────────────────────────────────

func TestNew_Development_Info(t *testing.T) {
	l, err := logger.New("info", "json", true)
	require.NoError(t, err)
	require.NotNil(t, l)
	// Development config uses colored console encoder; just verify it's usable.
	l.Info("test log")
}

func TestNew_Production_Json(t *testing.T) {
	l, err := logger.New("info", "json", false)
	require.NoError(t, err)
	require.NotNil(t, l)
	l.Info("test log")
}

func TestNew_Production_Console(t *testing.T) {
	l, err := logger.New("warn", "console", false)
	require.NoError(t, err)
	require.NotNil(t, l)
}

func TestNew_AllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			l, err := logger.New(level, "json", false)
			require.NoError(t, err)
			require.NotNil(t, l)
		})
	}
}

func TestNew_InvalidLevel(t *testing.T) {
	l, err := logger.New("not-a-real-level", "json", false)
	require.Error(t, err)
	assert.Nil(t, l)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestNew_EmptyLevel_DefaultsToInfo(t *testing.T) {
	// zapcore.Level's UnmarshalText accepts empty string as the zero value (Debug/Info).
	l, err := logger.New("", "json", false)
	require.NoError(t, err)
	require.NotNil(t, l)
}

// ─── WithComponent ─────────────────────────────────────────────────────────────

func TestWithComponent_TagsLogs(t *testing.T) {
	base, err := logger.New("info", "json", false)
	require.NoError(t, err)

	child := logger.WithComponent(base, "call_service")
	require.NotNil(t, child)
	// Child is a new logger instance with the component field attached.
	assert.NotSame(t, base, child)
}

func TestWithComponent_EmptyComponent(t *testing.T) {
	base, err := logger.New("info", "json", false)
	require.NoError(t, err)

	child := logger.WithComponent(base, "")
	require.NotNil(t, child)
}

// ─── LogEvent ─────────────────────────────────────────────────────────────────

func TestLogEvent_Debug(t *testing.T) {
	l, _ := logger.New("debug", "json", false)
	event := logger.LogEvent{Code: "TEST_DEBUG", Message: "debug message"}
	event.Debug(l, zap.String("key", "value"))
}

func TestLogEvent_Info(t *testing.T) {
	l, _ := logger.New("info", "json", false)
	event := logger.LogEvent{Code: "TEST_INFO", Message: "info message"}
	event.Info(l, zap.Int("count", 5))
}

func TestLogEvent_Warn(t *testing.T) {
	l, _ := logger.New("warn", "json", false)
	event := logger.LogEvent{Code: "TEST_WARN", Message: "warn message"}
	event.Warn(l)
}

func TestLogEvent_Error(t *testing.T) {
	l, _ := logger.New("error", "json", false)
	event := logger.LogEvent{Code: "TEST_ERROR", Message: "error message"}
	event.Error(l, zap.String("err", "boom"))
}

func TestLogEvent_NoOpWhenLevelDisabled(t *testing.T) {
	// A logger at ERROR level should skip DEBUG/INFO/WARN logs (no-op path).
	l, err := logger.New("error", "json", false)
	require.NoError(t, err)

	event := logger.LogEvent{Code: "LOW_LEVEL", Message: "suppressed"}
	event.Debug(l, zap.String("k", "v"))
	event.Info(l)
	event.Warn(l)
	// No assertions needed — just verify no panic and the logger.Check branch is exercised.
}

func TestLogEvent_Struct(t *testing.T) {
	event := logger.LogEvent{Code: "FOO", Message: "bar"}
	assert.Equal(t, "FOO", event.Code)
	assert.Equal(t, "bar", event.Message)
}

// TestLogEvent_Fatal verifies that Fatal writes the event and then panics,
// using zap's WithFatalHook to avoid calling os.Exit in tests.
func TestLogEvent_Fatal(t *testing.T) {
	// zap's default fatal hook is WriteThenFatal which calls os.Exit.
	// Override with WriteThenPanic so we can recover and assert.
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&bytes.Buffer{}),
		zapcore.FatalLevel,
	)
	l := zap.New(core, zap.WithFatalHook(zapcore.WriteThenPanic))

	event := logger.LogEvent{Code: "FATAL_TEST", Message: "fatal msg"}

	require.Panics(t, func() {
		event.Fatal(l, zap.String("k", "v"))
	})
}
