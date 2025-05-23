package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Level represents logging level
type Level = zerolog.Level

// Logger levels
const (
	DebugLevel = zerolog.DebugLevel
	InfoLevel  = zerolog.InfoLevel
	WarnLevel  = zerolog.WarnLevel
	ErrorLevel = zerolog.ErrorLevel
	FatalLevel = zerolog.FatalLevel
)

// Config holds logger configuration
type Config struct {
	Level      Level
	TimeFormat string
	Output     io.Writer
}

// Logger wraps zerolog.Logger
type Logger struct {
	ZL zerolog.Logger
}

// NewLogger creates a new logger instance
func NewLogger(cfg *Config) *Logger {
	if cfg == nil {
		cfg = &Config{
			Level:      InfoLevel,
			TimeFormat: time.RFC3339,
			Output:     os.Stdout,
		}
	}

	output := zerolog.ConsoleWriter{
		Out:        cfg.Output,
		TimeFormat: cfg.TimeFormat,
	}

	logger := zerolog.New(output).
		Level(cfg.Level).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{ZL: logger}
}

// WithContext adds context fields to logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{ZL: l.ZL.With().Interface("context", ctx).Logger()}
}

// WithFields adds fields to logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{ZL: l.ZL.With().Fields(fields).Logger()}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.ZL.Info().Fields(fields).Msg(msg)
}

func (l *Logger) Error(err error, msg string, fields ...interface{}) {
	l.ZL.Error().Err(err).Fields(fields).Msg(msg)
}

func (l *Logger) Fatal(err error, msg string, fields ...interface{}) {
	l.ZL.Fatal().Err(err).Fields(fields).Msg(msg)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.ZL.Debug().Fields(fields).Msg(msg)
}

// Add Warn method to Logger
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.ZL.Warn().Fields(fields).Msg(msg)
}
