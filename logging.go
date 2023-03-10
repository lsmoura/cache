package cache

import "context"

type Logger interface {
	Debug(msg string, params ...any)
	Info(msg string, params ...any)
	Error(msg string, params ...any)
}

type LoggerExtractor func(ctx context.Context) Logger

type logLevel int

const (
	logLevelDebug logLevel = iota
	logLevelInfo
	logLevelError
)

func (leve logLevel) String() string {
	switch leve {
	case logLevelDebug:
		return "DEBUG"
	case logLevelInfo:
		return "INFO"
	case logLevelError:
		return "ERROR"
	}

	return ""
}

type internalLogger struct {
	logger Logger

	keyvals []any
}

func (l *internalLogger) Debug(msg string, keyvals ...any) {
	l.logger.Debug(msg, append(l.keyvals, keyvals...)...)
}

func (l *internalLogger) Info(msg string, keyvals ...any) {
	l.logger.Info(msg, append(l.keyvals, keyvals...)...)
}

func (l *internalLogger) Error(msg string, keyvals ...any) {
	l.logger.Error(msg, append(l.keyvals, keyvals...)...)
}

func (l *internalLogger) With(keyvals ...any) *internalLogger {
	return &internalLogger{
		logger:  l.logger,
		keyvals: append(l.keyvals, keyvals...),
	}
}

type nilLoggerStruct struct{}

func (l *nilLoggerStruct) Debug(string, ...any) {}
func (l *nilLoggerStruct) Info(string, ...any)  {}
func (l *nilLoggerStruct) Error(string, ...any) {}
func (l *nilLoggerStruct) With(...any) Logger {
	return l
}

var nilLogger *nilLoggerStruct = nil

func (r Cache) log(ctx context.Context, level logLevel, msg string, keyvalues ...any) {
	if r.LogExtractor == nil {
		return
	}
	logger := r.LogExtractor(ctx)
	if logger == nil {
		return
	}

	switch level {
	case logLevelDebug:
		logger.Debug(msg, keyvalues...)
	case logLevelInfo:
		logger.Info(msg, keyvalues...)
	case logLevelError:
		logger.Error(msg, keyvalues...)
	default:
		logger.Info(msg, keyvalues...)
	}
}

func (r Cache) logger(ctx context.Context) Logger {
	if r.LogExtractor == nil {
		return nilLogger
	}
	logger := r.LogExtractor(ctx)
	if logger == nil {
		return nilLogger
	}

	return logger
}

func (r Cache) logDebug(ctx context.Context, msg string, keyvalues ...any) {
	r.log(ctx, logLevelDebug, msg, keyvalues...)
}

func (r Cache) logInfo(ctx context.Context, msg string, keyvalues ...any) {
	r.log(ctx, logLevelInfo, msg, keyvalues...)
}

func (r Cache) logError(ctx context.Context, msg string, keyvalues ...any) {
	r.log(ctx, logLevelError, msg, keyvalues...)
}
