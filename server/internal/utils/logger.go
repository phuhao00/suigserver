package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelFatal
)

var (
	currentLogLevel LogLevel = LevelInfo // Default log level
	logger                   = log.New(os.Stdout, "", 0) // Use a custom logger to control prefix and flags
)

func logLevelToString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// SetLogLevel sets the global log level for the application.
func SetLogLevel(levelString string) {
	switch strings.ToUpper(levelString) {
	case "DEBUG":
		currentLogLevel = LevelDebug
	case "INFO":
		currentLogLevel = LevelInfo
	case "WARNING", "WARN":
		currentLogLevel = LevelWarning
	case "ERROR":
		currentLogLevel = LevelError
	case "FATAL":
		currentLogLevel = LevelFatal
	default:
		currentLogLevel = LevelInfo
		LogWarnf("Unknown log level '%s', defaulting to INFO", levelString)
	}
	LogInfof("Log level set to %s", logLevelToString(currentLogLevel))
}

func logInternal(level LogLevel, message string) {
	if level >= currentLogLevel {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		// Using standard log package's Println to ensure atomic write for the whole line
		// and to match existing log output style more closely for now.
		// For more structured logging, a proper library would be better.
		log.Printf("%s [%s] %s\n", timestamp, logLevelToString(level), message)
	}
}

func LogDebug(args ...interface{}) {
	logInternal(LevelDebug, fmt.Sprint(args...))
}

func LogDebugf(format string, args ...interface{}) {
	logInternal(LevelDebug, fmt.Sprintf(format, args...))
}

func LogInfo(args ...interface{}) {
	logInternal(LevelInfo, fmt.Sprint(args...))
}

func LogInfof(format string, args ...interface{}) {
	logInternal(LevelInfo, fmt.Sprintf(format, args...))
}

func LogWarn(args ...interface{}) {
	logInternal(LevelWarning, fmt.Sprint(args...))
}

func LogWarnf(format string, args ...interface{}) {
	logInternal(LevelWarning, fmt.Sprintf(format, args...))
}

func LogError(args ...interface{}) {
	logInternal(LevelError, fmt.Sprint(args...))
}

func LogErrorf(format string, args ...interface{}) {
	logInternal(LevelError, fmt.Sprintf(format, args...))
}

func LogFatal(args ...interface{}) {
	logInternal(LevelFatal, fmt.Sprint(args...))
	os.Exit(1)
}

func LogFatalf(format string, args ...interface{}) {
	logInternal(LevelFatal, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// StandardLog returns a *log.Logger that respects the current log level,
// for use with libraries that expect a standard logger instance (e.g. actor system).
// Note: This is a simplified approach. The actor library's own logging might behave differently.
// For now, we're mostly replacing direct `log.Print` calls.
// This specific function might not be directly used if we replace `log.Print` calls.
func StandardLog() *log.Logger {
    // This is tricky because standard log.Logger doesn't have levels.
    // We'd need to pipe its output through our level filter, which is complex.
    // For now, it's better to replace log.Printf calls with our LogInfof etc.
    // Returning the base logger and letting users call its Printf is okay,
    // but it won't respect our levels unless we get very fancy.
	return logger // Returns the base logger; its direct use won't be level-filtered by our funcs.
}

// --- Proto.Actor Logger Bridge ---

// ProtoActorLogAdapter adapts our logger to the protoactor.Logger interface.
type ProtoActorLogAdapter struct{}

// Debug logs a Debug message.
func (l *ProtoActorLogAdapter) Debug(message string, args ...interface{}) {
	LogDebugf(message, args...)
}

// Info logs an Info message.
func (l *ProtoActorLogAdapter) Info(message string, args ...interface{}) {
	LogInfof(message, args...)
}

// Warning logs a Warning message.
func (l *ProtoActorLogAdapter) Warning(message string, args ...interface{}) {
	LogWarnf(message, args...)
}

// Error logs an Error message.
func (l *ProtoActorLogAdapter) Error(message string, args ...interface{}) {
	LogErrorf(message, args...)
}

// Fatal logs a Fatal message then exits.
func (l *ProtoActorLogAdapter) Fatal(message string, args ...interface{}) {
	LogFatalf(message, args...)
}
