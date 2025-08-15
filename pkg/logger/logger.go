package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kaanevranportfolio/Commercium/pkg/config"
)

// Logger wraps zap.SugaredLogger for structured logging
type Logger struct {
	*zap.SugaredLogger
	serviceName string
}

// New creates a new logger instance
func New(cfg config.LoggerConfig, serviceName string) (*Logger, error) {
	var zapConfig zap.Config

	// Configure based on environment
	switch cfg.Level {
	case "debug":
		zapConfig = zap.NewDevelopmentConfig()
	case "info", "warn", "error":
		zapConfig = zap.NewProductionConfig()
	default:
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Configure output
	if cfg.Output == "stdout" || cfg.Output == "" {
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	} else if cfg.Filename != "" {
		zapConfig.OutputPaths = []string{cfg.Filename}
		zapConfig.ErrorOutputPaths = []string{cfg.Filename}
	}

	// Configure encoding
	if cfg.Format == "json" || cfg.Format == "" {
		zapConfig.Encoding = "json"
	} else {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig = zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	}

	// Add service name to initial fields
	zapConfig.InitialFields = map[string]interface{}{
		"service": serviceName,
		"version": os.Getenv("APP_VERSION"),
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		SugaredLogger: logger.Sugar(),
		serviceName:   serviceName,
	}, nil
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields ...interface{}) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(fields...),
		serviceName:   l.serviceName,
	}
}

// WithCorrelationID adds correlation ID to logs
func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	return l.WithFields("correlation_id", correlationID)
}

// WithRequestID adds request ID to logs
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithFields("request_id", requestID)
}

// WithUserID adds user ID to logs
func (l *Logger) WithUserID(userID string) *Logger {
	return l.WithFields("user_id", userID)
}

// ServiceName returns the service name
func (l *Logger) ServiceName() string {
	return l.serviceName
}
