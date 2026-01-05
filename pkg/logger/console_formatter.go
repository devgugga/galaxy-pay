package logger

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// CustomLevelEncoder encodes log levels with colors and emojis for console
func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var emoji, color string
	switch level {
	case zapcore.DebugLevel:
		emoji = "🔵"
		color = "\033[36m" // Cyan
	case zapcore.InfoLevel:
		emoji = "🟢"
		color = "\033[32m" // Green
	case zapcore.WarnLevel:
		emoji = "🟡"
		color = "\033[33m" // Yellow
	case zapcore.ErrorLevel:
		emoji = "🔴"
		color = "\033[31m" // Red
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		emoji = "💥"
		color = "\033[35m" // Magenta
	default:
		emoji = "⚪"
		color = "\033[37m" // White
	}
	reset := "\033[0m"
	enc.AppendString(color + emoji + " " + level.CapitalString() + reset)
}

// CustomTimeEncoder encodes time in a readable format
func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

