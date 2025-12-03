package pkg

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestSetLogLevel(t *testing.T) {
	original := GetLogLevel()
	defer SetLogLevel(original)

	tests := []struct {
		name  string
		level slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLogLevel(tt.level)
			if got := GetLogLevel(); got != tt.level {
				t.Errorf("GetLogLevel() = %v, want %v", got, tt.level)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	logger.Info("test message")
	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("log output missing message: %s", buf.String())
	}
}

func TestNewJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	if logger == nil {
		t.Fatal("NewJSONLogger returned nil")
	}

	logger.Info("test message")
	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("JSON log output missing message: %s", output)
	}
}

func TestLogDebug(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogLevel(slog.LevelDebug)
	SetLogger(NewLogger(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	LogDebug(ComponentDevice, "debug message", "key", "value")
	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Errorf("debug log missing message: %s", output)
	}
	if !strings.Contains(output, "component=device") {
		t.Errorf("debug log missing component: %s", output)
	}
}

func TestLogInfo(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(&buf, nil))

	LogInfo(ComponentHost, "info message")
	output := buf.String()
	if !strings.Contains(output, "info message") {
		t.Errorf("info log missing message: %s", output)
	}
	if !strings.Contains(output, "component=host") {
		t.Errorf("info log missing component: %s", output)
	}
}

func TestLogWarn(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(&buf, nil))

	LogWarn(ComponentStack, "warn message")
	output := buf.String()
	if !strings.Contains(output, "warn message") {
		t.Errorf("warn log missing message: %s", output)
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(&buf, nil))

	LogError(ComponentHAL, "error message")
	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("error log missing message: %s", output)
	}
}

func TestSetLogger(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	customLogger := NewLogger(&buf, nil)
	SetLogger(customLogger)

	LogInfo(ComponentDevice, "custom logger test")
	if !strings.Contains(buf.String(), "custom logger test") {
		t.Error("custom logger not used")
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestComponentString tests Component string conversion
func TestComponentString(t *testing.T) {
	components := []Component{
		ComponentDevice,
		ComponentHost,
		ComponentStack,
		ComponentHAL,
		ComponentTransfer,
		ComponentEndpoint,
	}

	for _, c := range components {
		if string(c) == "" {
			t.Errorf("Component %v has empty string", c)
		}
	}
}

// TestLogWithEmptyArgs tests log functions with no extra args
func TestLogWithEmptyArgs(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogLevel(slog.LevelDebug)
	SetLogger(NewLogger(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	LogDebug(ComponentDevice, "empty args test")
	output := buf.String()
	if !strings.Contains(output, "empty args test") {
		t.Errorf("log missing message: %s", output)
	}
}

// TestLogWithManyArgs tests log functions with many key-value pairs
func TestLogWithManyArgs(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(&buf, nil))

	LogInfo(ComponentDevice, "many args",
		"key1", "value1",
		"key2", 42,
		"key3", true,
		"key4", 3.14,
	)
	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("log missing key1: %s", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("log missing key2: %s", output)
	}
}

// TestLogLevelFiltering tests that log levels are respected
func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	original := DefaultLogger
	originalLevel := GetLogLevel()
	defer func() {
		DefaultLogger = original
		SetLogLevel(originalLevel)
	}()

	SetLogLevel(slog.LevelWarn)
	SetLogger(NewLogger(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// These should not appear
	LogDebug(ComponentDevice, "debug should not appear")
	LogInfo(ComponentDevice, "info should not appear")

	// These should appear
	LogWarn(ComponentDevice, "warn should appear")
	LogError(ComponentDevice, "error should appear")

	output := buf.String()
	if strings.Contains(output, "debug should not appear") {
		t.Error("debug message appeared when level was Warn")
	}
	if strings.Contains(output, "info should not appear") {
		t.Error("info message appeared when level was Warn")
	}
	if !strings.Contains(output, "warn should appear") {
		t.Error("warn message did not appear")
	}
	if !strings.Contains(output, "error should appear") {
		t.Error("error message did not appear")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkSetLogLevel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetLogLevel(slog.LevelInfo)
	}
}

func BenchmarkGetLogLevel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetLogLevel()
	}
}

func BenchmarkNewLogger(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewLogger(io.Discard, nil)
	}
}

func BenchmarkNewJSONLogger(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewJSONLogger(io.Discard, nil)
	}
}

// discardLogger returns a logger that discards all output
func discardLogger() *slog.Logger {
	return NewLogger(io.Discard, nil)
}

func BenchmarkLogDebug_Enabled(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogLevel(slog.LevelDebug)
	SetLogger(NewLogger(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogDebug(ComponentDevice, "test message", "key", "value")
	}
}

func BenchmarkLogDebug_Disabled(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogLevel(slog.LevelInfo) // Debug disabled
	SetLogger(NewLogger(io.Discard, nil))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogDebug(ComponentDevice, "test message", "key", "value")
	}
}

func BenchmarkLogInfo(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(io.Discard, nil))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogInfo(ComponentDevice, "test message", "key", "value")
	}
}

func BenchmarkLogWarn(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(io.Discard, nil))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogWarn(ComponentDevice, "test message", "key", "value")
	}
}

func BenchmarkLogError(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(io.Discard, nil))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogError(ComponentDevice, "test message", "key", "value")
	}
}

func BenchmarkLogInfo_ManyArgs(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(io.Discard, nil))

	argCounts := []int{0, 2, 4, 8}
	for _, n := range argCounts {
		args := make([]any, 0, n)
		for i := 0; i < n; i += 2 {
			args = append(args, "key", "value")
		}
		b.Run(strings.ReplaceAll(strings.Repeat("kv", n/2), "kv", "kv"), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				LogInfo(ComponentDevice, "test message", args...)
			}
		})
	}
}

func BenchmarkLogInfo_AllComponents(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewLogger(io.Discard, nil))

	components := []Component{
		ComponentDevice,
		ComponentHost,
		ComponentStack,
		ComponentHAL,
		ComponentTransfer,
		ComponentEndpoint,
	}

	for _, c := range components {
		b.Run(string(c), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				LogInfo(c, "test message")
			}
		})
	}
}

func BenchmarkLogJSON(b *testing.B) {
	original := DefaultLogger
	defer func() { DefaultLogger = original }()

	SetLogger(NewJSONLogger(io.Discard, nil))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogInfo(ComponentDevice, "test message", "key", "value")
	}
}
