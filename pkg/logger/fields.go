package logger

import "go.uber.org/zap"

// RequestID returns a zap field for request ID
func RequestID(id string) zap.Field {
	return zap.String("request_id", id)
}

// Method returns a zap field for HTTP method
func Method(method string) zap.Field {
	return zap.String("method", method)
}

// Path returns a zap field for HTTP path
func Path(path string) zap.Field {
	return zap.String("path", path)
}

// Status returns a zap field for HTTP status code
func Status(status int) zap.Field {
	return zap.Int("status", status)
}

// Latency returns a zap field for request latency
func Latency(latency string) zap.Field {
	return zap.String("latency", latency)
}

// IP returns a zap field for client IP
func IP(ip string) zap.Field {
	return zap.String("ip", ip)
}

// UserAgent returns a zap field for user agent
func UserAgent(ua string) zap.Field {
	return zap.String("user_agent", ua)
}

// Error returns a zap field for error
func Error(err error) zap.Field {
	return zap.Error(err)
}

// String returns a zap field for string value
func String(key, value string) zap.Field {
	return zap.String(key, value)
}

// Int returns a zap field for int value
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Float64 returns a zap field for float64 value
func Float64(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

// Bool returns a zap field for bool value
func Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

// Duration returns a zap field for duration
func Duration(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

