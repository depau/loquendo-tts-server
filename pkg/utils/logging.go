package utils

import (
	"fmt"

	"github.com/rs/zerolog"
)

func SetLogLevel(level string) error {
	var logLevel zerolog.Level
	switch level {
	case "trace":
		logLevel = zerolog.TraceLevel
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	case "fatal":
		logLevel = zerolog.FatalLevel
	case "panic":
		logLevel = zerolog.PanicLevel
	default:
		return fmt.Errorf("invalid log level: %s", level)
	}
	zerolog.SetGlobalLevel(logLevel)
	return nil
}
