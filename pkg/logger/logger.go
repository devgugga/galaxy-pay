package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance
var Log *zap.Logger

// Init initializes the logger based on the environment
func Init(env string, logLevel string) error {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.MessageKey = "message"
		config.EncoderConfig.LevelKey = "level"
		config.EncoderConfig.CallerKey = "caller"
		config.EncoderConfig.StacktraceKey = "stacktrace"
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = CustomLevelEncoder
		config.EncoderConfig.EncodeTime = CustomTimeEncoder
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	// Set log level
	level, _ := parseLogLevel(logLevel)
	config.Level = zap.NewAtomicLevelAt(level)

	// Build logger
	var buildErr error
	Log, buildErr = config.Build()
	if buildErr != nil {
		return buildErr
	}

	// Replace global logger
	zap.ReplaceGlobals(Log)

	return nil
}

// parseLogLevel parses a string log level to zapcore.Level
func parseLogLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, nil
	}
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// WithRequestID creates a logger with request ID field
func WithRequestID(requestID string) *zap.Logger {
	if Log == nil {
		return zap.NewNop()
	}
	return Log.With(zap.String("request_id", requestID))
}

// WithFields creates a logger with multiple fields
func WithFields(fields ...zap.Field) *zap.Logger {
	if Log == nil {
		return zap.NewNop()
	}
	return Log.With(fields...)
}

// IsDevelopment returns true if running in development mode
func IsDevelopment() bool {
	return os.Getenv("APP_ENV") != "production"
}

