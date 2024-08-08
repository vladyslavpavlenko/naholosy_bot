package logger

import (
	"os"
	"time"

	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger/rotator"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DebugLevel = "DEBUG"
	ProdLevel  = "PROD"
)

// A Logger represents an active logging object.
type Logger struct {
	z *zap.Logger
}

// Get return a pointer the underlying zap.Logger.
func (l *Logger) Get() *zap.Logger {
	return l.z
}

// New creates and returns a new logger.
func New(lvl string) *Logger {
	// Initialize zap config.
	config := newZapConfig(lvl)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// Initialize zap logging core.
	sync := zapcore.AddSync(zapcore.Lock(os.Stdout))
	enc := zapcore.NewConsoleEncoder(config.EncoderConfig)
	core := zapcore.NewCore(enc, sync, config.Level)

	z := zap.New(core)

	return &Logger{
		z: z,
	}
}

// NewWithRotation method sets up logger that writes logs to the specified writer and file
// simultaneously. This logger automatically rotates log files with the options provided.
func NewWithRotation(lvl string, opts *rotator.Options) *Logger {
	// Initialize zap logging core.
	zConfig := newZapConfig(lvl)
	zConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.StampMilli)
	zConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zSync := zapcore.AddSync(zapcore.Lock(os.Stdout))
	zEnc := zapcore.NewConsoleEncoder(zConfig.EncoderConfig)
	zCore := zapcore.NewCore(zEnc, zSync, zConfig.Level)

	// Initialize rotator logging core.
	rConfig := newZapConfig(lvl)
	rConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.StampMilli)
	rConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	rSync := zapcore.AddSync(rotator.New(opts).Logger)
	rEnc := zapcore.NewConsoleEncoder(rConfig.EncoderConfig)
	rCore := zapcore.NewCore(rEnc, rSync, rConfig.Level)

	// Combine two loggers.
	z := zap.New(zapcore.NewTee(zCore, rCore))

	return &Logger{
		z: z,
	}
}

// newZapConfig returns new zap.Config depending on the logging level provided.
func newZapConfig(lvl string) zap.Config {
	switch lvl {
	case DebugLevel:
		return zap.NewDevelopmentConfig()
	case ProdLevel:
		return zap.NewProductionConfig()
	default:
		return zap.NewDevelopmentConfig()
	}
}

// With creates a child logger and adds structured context to it.
func (l *Logger) With(args ...zap.Field) *Logger {
	l.z = l.z.With(args...)
	return l
}

// Debug logs a message at Debug level.
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.z.Debug(msg, fields...)
}

// Info logs a message at Info level.
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.z.Info(msg, fields...)
}

// Warn logs a message at Warn level.
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.z.Warn(msg, fields...)
}

// Error logs a message at Error level.
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.z.Error(msg, fields...)
}

// Fatal logs a message at Fatal level.
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.z.Fatal(msg, fields...)
}
