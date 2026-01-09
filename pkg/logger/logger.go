package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
)

// Level represents log level
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

var levelColors = map[Level]func(format string, a ...interface{}) string{
	DebugLevel: color.CyanString,
	InfoLevel:  color.GreenString,
	WarnLevel:  color.YellowString,
	ErrorLevel: color.RedString,
}

// Logger is the main logger struct
type Logger struct {
	level  Level
	output io.Writer
}

var defaultLogger = &Logger{
	level:  InfoLevel,
	output: os.Stderr,
}

// SetLevel sets the log level
func SetLevel(level Level) {
	defaultLogger.level = level
}

// SetOutput sets the output writer
func SetOutput(w io.Writer) {
	defaultLogger.output = w
}

// EnableDebug enables debug logging
func EnableDebug() {
	SetLevel(DebugLevel)
}

func log(level Level, format string, args ...interface{}) {
	if level < defaultLogger.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := levelColors[level]("[%s]", levelNames[level])
	message := fmt.Sprintf(format, args...)

	fmt.Fprintf(defaultLogger.output, "%s %s %s\n", timestamp, levelStr, message)
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	log(DebugLevel, format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	log(InfoLevel, format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	log(WarnLevel, format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	log(ErrorLevel, format, args...)
}

// Success prints a success message (green checkmark)
func Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(defaultLogger.output, "%s %s\n", color.GreenString("✓"), message)
}

// Failure prints a failure message (red X)
func Failure(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(defaultLogger.output, "%s %s\n", color.RedString("✗"), message)
}

// Bold prints a bold message
func Bold(format string, args ...interface{}) string {
	return color.New(color.Bold).Sprintf(format, args...)
}
