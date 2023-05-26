package logger

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/instill-ai/pipeline-backend/config"
)

var logger *zap.Logger
var once sync.Once
var core zapcore.Core

// GetZapLogger returns an instance of zap logger
func GetZapLogger(ctx context.Context) (*zap.Logger, error) {
	var err error
	once.Do(func() {
		// debug and info level enabler
		debugInfoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.DebugLevel || level == zapcore.InfoLevel
		})

		// info level enabler
		infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.InfoLevel
		})

		// warn, error and fatal level enabler
		warnErrorFatalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.WarnLevel || level == zapcore.ErrorLevel || level == zapcore.FatalLevel
		})

		// write syncers
		stdoutSyncer := zapcore.Lock(os.Stdout)
		stderrSyncer := zapcore.Lock(os.Stderr)

		// tee core
		if config.Config.Server.Debug {
			core = zapcore.NewTee(
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
					stdoutSyncer,
					debugInfoLevel,
				),
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
					stderrSyncer,
					warnErrorFatalLevel,
				),
			)
		} else {
			core = zapcore.NewTee(
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
					stdoutSyncer,
					infoLevel,
				),
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
					stderrSyncer,
					warnErrorFatalLevel,
				),
			)
		}

		// finally construct the logger with the tee core
		logger = zap.New(core)
	})

	// hooks to inject logs to traces
	logger = logger.WithOptions(
		zap.Hooks(func(entry zapcore.Entry) error {
			span := trace.SpanFromContext(ctx)
			if !span.IsRecording() {
				return nil
			}

			attrs := make([]attribute.KeyValue, 0)
			logSeverityKey := attribute.Key("log.severity")
			logMessageKey := attribute.Key("log.message")
			attrs = append(attrs, logSeverityKey.String(entry.Level.String()))
			attrs = append(attrs, logMessageKey.String(entry.Message))

			span.AddEvent("log", trace.WithAttributes(attrs...))
			if entry.Level >= zap.ErrorLevel {
				span.SetStatus(codes.Error, string(entry.Message))
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return nil
		}))

	return logger, err
}
