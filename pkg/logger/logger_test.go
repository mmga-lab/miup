package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLevelConstants(t *testing.T) {
	if DebugLevel >= InfoLevel {
		t.Error("DebugLevel should be less than InfoLevel")
	}
	if InfoLevel >= WarnLevel {
		t.Error("InfoLevel should be less than WarnLevel")
	}
	if WarnLevel >= ErrorLevel {
		t.Error("WarnLevel should be less than ErrorLevel")
	}
}

func TestLevelNames(t *testing.T) {
	tests := []struct {
		level Level
		name  string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
	}

	for _, tt := range tests {
		if levelNames[tt.level] != tt.name {
			t.Errorf("levelNames[%d] = %s, want %s", tt.level, levelNames[tt.level], tt.name)
		}
	}
}

func TestSetLevel(t *testing.T) {
	originalLevel := defaultLogger.level
	defer SetLevel(originalLevel)

	SetLevel(DebugLevel)
	if defaultLogger.level != DebugLevel {
		t.Errorf("SetLevel(DebugLevel) did not set level correctly")
	}

	SetLevel(ErrorLevel)
	if defaultLogger.level != ErrorLevel {
		t.Errorf("SetLevel(ErrorLevel) did not set level correctly")
	}
}

func TestSetOutput(t *testing.T) {
	originalOutput := defaultLogger.output
	defer SetOutput(originalOutput)

	buf := &bytes.Buffer{}
	SetOutput(buf)
	if defaultLogger.output != buf {
		t.Error("SetOutput did not set output correctly")
	}
}

func TestEnableDebug(t *testing.T) {
	originalLevel := defaultLogger.level
	defer SetLevel(originalLevel)

	EnableDebug()
	if defaultLogger.level != DebugLevel {
		t.Error("EnableDebug did not set level to DebugLevel")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	originalLevel := defaultLogger.level
	originalOutput := defaultLogger.output
	defer func() {
		SetLevel(originalLevel)
		SetOutput(originalOutput)
	}()

	buf := &bytes.Buffer{}
	SetOutput(buf)

	// Set level to WarnLevel, Debug and Info should be filtered
	SetLevel(WarnLevel)

	Debug("debug message")
	Info("info message")
	if buf.Len() > 0 {
		t.Error("Debug and Info should be filtered when level is WarnLevel")
	}

	Warn("warn message")
	if !strings.Contains(buf.String(), "WARN") {
		t.Error("Warn should not be filtered when level is WarnLevel")
	}

	buf.Reset()
	Error("error message")
	if !strings.Contains(buf.String(), "ERROR") {
		t.Error("Error should not be filtered when level is WarnLevel")
	}
}

func TestDebug(t *testing.T) {
	originalLevel := defaultLogger.level
	originalOutput := defaultLogger.output
	defer func() {
		SetLevel(originalLevel)
		SetOutput(originalOutput)
	}()

	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(DebugLevel)

	Debug("test %s", "message")
	output := buf.String()

	if !strings.Contains(output, "DEBUG") {
		t.Error("Debug output should contain DEBUG")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Debug output should contain formatted message")
	}
}

func TestInfo(t *testing.T) {
	originalLevel := defaultLogger.level
	originalOutput := defaultLogger.output
	defer func() {
		SetLevel(originalLevel)
		SetOutput(originalOutput)
	}()

	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(InfoLevel)

	Info("test %s", "message")
	output := buf.String()

	if !strings.Contains(output, "INFO") {
		t.Error("Info output should contain INFO")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Info output should contain formatted message")
	}
}

func TestWarn(t *testing.T) {
	originalLevel := defaultLogger.level
	originalOutput := defaultLogger.output
	defer func() {
		SetLevel(originalLevel)
		SetOutput(originalOutput)
	}()

	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(WarnLevel)

	Warn("test %s", "warning")
	output := buf.String()

	if !strings.Contains(output, "WARN") {
		t.Error("Warn output should contain WARN")
	}
	if !strings.Contains(output, "test warning") {
		t.Error("Warn output should contain formatted message")
	}
}

func TestError(t *testing.T) {
	originalLevel := defaultLogger.level
	originalOutput := defaultLogger.output
	defer func() {
		SetLevel(originalLevel)
		SetOutput(originalOutput)
	}()

	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(ErrorLevel)

	Error("test %s", "error")
	output := buf.String()

	if !strings.Contains(output, "ERROR") {
		t.Error("Error output should contain ERROR")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Error output should contain formatted message")
	}
}

func TestSuccess(t *testing.T) {
	originalOutput := defaultLogger.output
	defer SetOutput(originalOutput)

	buf := &bytes.Buffer{}
	SetOutput(buf)

	Success("operation %s", "completed")
	output := buf.String()

	if !strings.Contains(output, "operation completed") {
		t.Error("Success output should contain formatted message")
	}
}

func TestFailure(t *testing.T) {
	originalOutput := defaultLogger.output
	defer SetOutput(originalOutput)

	buf := &bytes.Buffer{}
	SetOutput(buf)

	Failure("operation %s", "failed")
	output := buf.String()

	if !strings.Contains(output, "operation failed") {
		t.Error("Failure output should contain formatted message")
	}
}

func TestBold(t *testing.T) {
	result := Bold("test %s", "message")
	if !strings.Contains(result, "test message") {
		t.Error("Bold should return formatted message")
	}
}

func TestLogTimestamp(t *testing.T) {
	originalLevel := defaultLogger.level
	originalOutput := defaultLogger.output
	defer func() {
		SetLevel(originalLevel)
		SetOutput(originalOutput)
	}()

	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(InfoLevel)

	Info("test")
	output := buf.String()

	// Check timestamp format (YYYY-MM-DD HH:MM:SS)
	if len(output) < 19 {
		t.Error("Log output should contain timestamp")
	}
	// Simple check for timestamp format (should start with year)
	if output[0] != '2' {
		t.Errorf("Log output should start with timestamp year, got: %s", output[:20])
	}
}
