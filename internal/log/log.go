// Package log provides a global logger for zap.
package log

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Zap global logger, log.Sync() must be called before exiting.
var log *zap.Logger

func init() {
	// Initialize to No-Op to avoid panics
	log = zap.NewNop()
}

// Setup configures zap's logger accordingly to the parameters received.
func Setup(development bool, outFiles []string) error {
	var (
		config zap.Config
		err    error
	)

	if development {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	for _, f := range outFiles {
		dir := filepath.Dir(f)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return errors.Wrapf(err, "creating directories: %q", f)
			}
		}
	}
	config.OutputPaths = append(config.OutputPaths, outFiles...)

	log, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return errors.Wrap(err, "building logger configuration")
	}

	return nil
}

// Check returns a CheckedEntry if logging a message at the specified level
// is enabled. It's a completely optional optimization; in high-performance
// applications, Check can help avoid allocating a slice to hold fields.
func Check(lvl zapcore.Level, msg string) *zapcore.CheckedEntry {
	return log.Check(lvl, msg)
}

// Core returns the Logger's underlying zapcore.Core.
func Core() zapcore.Core {
	return log.Core()
}

// DPanic logs a message at DPanicLevel. The message includes any fields
// passed at the log site, as well as any fields accumulated on the logger.
//
// If the logger is in development mode, it then panics (DPanic means
// "development panic"). This is useful for catching errors that are
// recoverable, but shouldn't ever happen.
func DPanic(msg string, fields ...zap.Field) {
	log.DPanic(msg, fields...)
}

// Debug logs a message at DebugLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

// Fatal logs a message at FatalLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then calls os.Exit(1), even if logging at FatalLevel is
// disabled.
func Fatal(msg string, fields ...zap.Field) {
	log.Fatal(msg, fields...)
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

// Named adds a new path segment to the logger's name. Segments are joined by
// periods. By default, Loggers are unnamed.
func Named(s string) *zap.Logger {
	return log.Named(s)
}

// Panic logs a message at PanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then panics, even if logging at PanicLevel is disabled.
func Panic(msg string, fields ...zap.Field) {
	log.Panic(msg, fields...)
}

// Sugar wraps the Logger to provide a more ergonomic, but slightly slower,
// API. Sugaring a Logger is quite inexpensive, so it's reasonable for a
// single application to use both Loggers and SugaredLoggers, converting
// between them on the boundaries of performance-sensitive code.
func Sugar() *zap.SugaredLogger {
	return log.Sugar()
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func Sync() error {
	return log.Sync()
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

// With creates a child logger and adds structured context to it. Fields added
// to the child don't affect the parent, and vice versa.
func With(fields ...zap.Field) *zap.Logger {
	return log.With(fields...)
}

// WithOptions clones the current Logger, applies the supplied Options, and
// returns the resulting Logger. It's safe to use concurrently.
func WithOptions(opts ...zap.Option) *zap.Logger {
	return log.WithOptions(opts...)
}
