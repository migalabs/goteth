package utils

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// App default configurations
var (
	ModName = "utils"
	log     = logrus.WithField(
		"module", ModName,
	)
	DefaultLoglvl    = logrus.InfoLevel
	DefaultLogOutput = os.Stdout
	DefaultFormater  = &logrus.TextFormatter{}
)

// Select Log Level from string
func ParseLogLevel(lvl string) logrus.Level {
	switch lvl {
	case "trace":
		return logrus.TraceLevel
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return DefaultLoglvl
	}
}

// parse Formatter from string
func ParseLogOutput(lvl string) io.Writer {
	switch lvl {
	case "terminal":
		return os.Stdout
	default:
		return DefaultLogOutput
	}
}

// parse Formatter from string
func ParseLogFormatter(lvl string) logrus.Formatter {
	switch lvl {
	case "text":
		return &logrus.TextFormatter{}
	default:
		return DefaultFormater
	}
}
