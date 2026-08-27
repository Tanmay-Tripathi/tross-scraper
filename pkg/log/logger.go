// Package log provides the context-aware, structured logger used across the
// service. It wraps zerolog and automatically decorates every entry with the
// request and correlation IDs carried on the context.
package log

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"github.com/rs/zerolog"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
)

// Logger is a logger that supports log levels, context and structured logging.
type Logger interface {
	// With returns a logger derived from the receiver, decorated with the
	// trace identifiers on ctx plus any additional key/value argument pairs.
	With(ctx context.Context, args ...any) Logger

	// Debug uses fmt.Sprint to construct and log a message at DEBUG level.
	Debug(args ...any)
	// Info uses fmt.Sprint to construct and log a message at INFO level.
	Info(args ...any)
	// Warn uses fmt.Sprint to construct and log a message at WARN level.
	Warn(args ...any)
	// Error uses fmt.Sprint to construct and log a message at ERROR level.
	Error(args ...any)

	// Debugf uses fmt.Sprintf to construct and log a message at DEBUG level.
	Debugf(format string, args ...any)
	// Infof uses fmt.Sprintf to construct and log a message at INFO level.
	Infof(format string, args ...any)
	// Warnf uses fmt.Sprintf to construct and log a message at WARN level.
	Warnf(format string, args ...any)
	// Errorf uses fmt.Sprintf to construct and log a message at ERROR level.
	Errorf(format string, args ...any)

	// WithRequest returns a context that knows the request and correlation IDs
	// carried by req, generating a request ID when the caller did not send one.
	WithRequest(ctx context.Context, req *http.Request) context.Context
}

// LogConfig describes the static fields stamped onto every log entry.
type LogConfig struct {
	ServiceName string
	AppEnv      string
	AppVersion  string
	// Level is an optional zerolog level name ("debug", "info", "warn",
	// "error"). It defaults to "info" when empty or unrecognised.
	Level string
}

type logger struct {
	log zerolog.Logger
}

// New creates the root logger. Local runs get human-readable console output;
// every other environment gets structured JSON on stdout.
func New(cfg LogConfig) Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	var writer io.Writer = os.Stdout
	if cfg.AppEnv == string(global.LocalEnv) {
		writer = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	zl := zerolog.New(writer).
		Level(parseLevel(cfg.Level)).
		With().
		Timestamp().
		Str("v", cfg.AppVersion).
		Str("env", cfg.AppEnv).
		Str("service", cfg.ServiceName).
		CallerWithSkipFrameCount(3).
		Logger()

	return &logger{log: zl}
}

// NewWithZerolog wraps a preconfigured zerolog logger, mainly for tests.
func NewWithZerolog(zl zerolog.Logger) Logger {
	return &logger{log: zl}
}

func parseLevel(level string) zerolog.Level {
	parsed, err := zerolog.ParseLevel(level)
	if err != nil || parsed == zerolog.NoLevel {
		return zerolog.InfoLevel
	}
	return parsed
}

func (l *logger) With(ctx context.Context, args ...any) Logger {
	if _, ok := ctx.(*gin.Context); ok {
		l.Warn("use context.Context instead of *gin.Context so tracing fields propagate")
	}

	if ctx == nil && len(args) == 0 {
		return l
	}

	logCtx := l.log.With()

	if ctx != nil {
		if id := global.RequestIDFromContext(ctx); id != "" {
			logCtx = logCtx.Str(global.RequestID.String(), id)
		}
		if id := global.CorrelationIDFromContext(ctx); id != "" {
			logCtx = logCtx.Str(global.CorrelationID.String(), id)
		}
	}

	// args are key/value pairs; a trailing odd argument is ignored.
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", args[i])
		}
		logCtx = logCtx.Interface(key, args[i+1])
	}

	return &logger{log: logCtx.Logger()}
}

func (l *logger) WithRequest(ctx context.Context, req *http.Request) context.Context {
	requestID := req.Header.Get(global.RequestID.String())
	if requestID == "" {
		requestID = req.Header.Get(global.XRequestID.String())
	}
	if requestID == "" {
		requestID = xid.New().String()
	}

	correlationID := req.Header.Get(global.CorrelationID.String())
	if correlationID == "" {
		correlationID = xid.New().String()
	}

	ctx = global.WithRequestID(ctx, requestID)
	return global.WithCorrelationID(ctx, correlationID)
}

func (l *logger) Debug(args ...any) { l.log.Debug().Msg(fmt.Sprint(args...)) }
func (l *logger) Info(args ...any)  { l.log.Info().Msg(fmt.Sprint(args...)) }
func (l *logger) Warn(args ...any)  { l.log.Warn().Msg(fmt.Sprint(args...)) }
func (l *logger) Error(args ...any) { l.log.Error().Msg(fmt.Sprint(args...)) }

func (l *logger) Debugf(format string, args ...any) { l.log.Debug().Msgf(format, args...) }
func (l *logger) Infof(format string, args ...any)  { l.log.Info().Msgf(format, args...) }
func (l *logger) Warnf(format string, args ...any)  { l.log.Warn().Msgf(format, args...) }
func (l *logger) Errorf(format string, args ...any) { l.log.Error().Msgf(format, args...) }
