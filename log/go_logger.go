package log

import (
	"log"
	"os"
)

type goLogger struct {
	logger   *log.Logger
	logLevel int
}

const (
	DebugLevel int = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
	FatalLevel
)

func getLogLevel(level string) int {
	switch level {
	case Info:
		return InfoLevel
	case Warn:
		return WarnLevel
	case Debug:
		return DebugLevel
	case Error:
		return ErrorLevel
	case Fatal:
		return FatalLevel
	default:
		return InfoLevel
	}
}

func newGoLogger(config Configuration) (Logger, error) {
	return &goLogger{
		logger:   log.New(os.Stderr, "", log.LstdFlags),
		logLevel: getLogLevel(config.ConsoleLevel),
	}, nil
}

func (l *goLogger) Debugf(format string, args ...interface{}) {
	if l.logLevel > DebugLevel {
		return
	}
	l.logger.Printf(format, args...)
}

func (l *goLogger) Infof(format string, args ...interface{}) {
	if l.logLevel > InfoLevel {
		return
	}
	l.logger.Printf(format, args...)
}

func (l *goLogger) Warnf(format string, args ...interface{}) {
	if l.logLevel > WarnLevel {
		return
	}
	l.logger.Printf(format, args...)
}

func (l *goLogger) Errorf(format string, args ...interface{}) {
	if l.logLevel > ErrorLevel {
		return
	}
	l.logger.Printf(format, args...)
}

func (l *goLogger) Fatalf(format string, args ...interface{}) {
	if l.logLevel > FatalLevel {
		return
	}
	l.logger.Fatalf(format, args...)
}

func (l *goLogger) Panicf(format string, args ...interface{}) {
	if l.logLevel > PanicLevel {
		return
	}
	l.logger.Panicf(format, args...)
}

func (l *goLogger) WithFields(fields Fields) Logger {
	// not implemented
	return l
}
