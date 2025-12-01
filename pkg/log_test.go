package pkg

import (
	"bytes"
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
	logger := NewLogger(&buf, nil)
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
	logger := NewJSONLogger(&buf, nil)
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
