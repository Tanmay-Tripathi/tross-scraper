// Package telemetry wires OTel tracing and Prometheus metrics into the router and
// stamps a request and correlation ID onto every request.
package telemetry

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/xid"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// Config describes the telemetry pipeline for one service instance.
type Config struct {
	Logger      log.Logger
	ServiceName string
	AppEnv      string
	AppVersion  string
	// ExporterURL is the OTLP/HTTP endpoint; empty disables tracing.
	ExporterURL string
}

// Methods is the telemetry surface the app wiring depends on.
type Methods interface {
	// EnableGinTracing installs the middleware and exposes /metrics.
	EnableGinTracing(engine *gin.Engine)
	// Shutdown flushes any buffered spans.
	Shutdown(ctx context.Context) error
}

type telemetry struct {
	cfg      Config
	metrics  *metrics
	shutdown func(context.Context) error
}

// New initialises the trace provider and metric collectors.
func New(cfg Config) Methods {
	t := &telemetry{
		cfg:     cfg,
		metrics: newMetrics(cfg.ServiceName),
	}
	t.shutdown = t.initTracing()
	return t
}

func (t *telemetry) initTracing() func(context.Context) error {
	logger := t.cfg.Logger

	if t.cfg.ExporterURL == "" {
		logger.Infof("no OTLP exporter configured, tracing disabled")
		return nil
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(t.cfg.ExporterURL),
		otlptracehttp.WithInsecure(),
	)

	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		logger.Errorf("failed to initialize OTLP trace exporter: %v", err)
		return nil
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(t.resource()),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Trace outbound HTTP calls made through the default transport too.
	http.DefaultTransport = otelhttp.NewTransport(http.DefaultTransport)

	logger.Infof("tracing enabled, exporting to %s", t.cfg.ExporterURL)
	return provider.Shutdown
}

func (t *telemetry) resource() *sdkresource.Resource {
	detected, _ := sdkresource.New(
		context.Background(),
		sdkresource.WithOS(),
		sdkresource.WithProcess(),
		sdkresource.WithContainer(),
		sdkresource.WithHost(),
		sdkresource.WithSchemaURL(semconv.SchemaURL),
		sdkresource.WithAttributes(
			attribute.String("service.name", t.cfg.ServiceName),
			attribute.String("service.version", t.cfg.AppVersion),
			attribute.String("deployment.environment.name", t.cfg.AppEnv),
		),
	)

	merged, err := sdkresource.Merge(sdkresource.Default(), detected)
	if err != nil {
		t.cfg.Logger.Warnf("failed to merge telemetry resource attributes: %v", err)
		return sdkresource.Default()
	}
	return merged
}

func (t *telemetry) EnableGinTracing(engine *gin.Engine) {
	engine.Use(TraceIDMiddleware())
	engine.Use(otelgin.Middleware(t.cfg.ServiceName))
	engine.Use(t.metrics.middleware())
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func (t *telemetry) Shutdown(ctx context.Context) error {
	if t.shutdown == nil {
		return nil
	}
	return t.shutdown(ctx)
}

// TraceIDMiddleware puts a request and correlation ID on the context, so every
// downstream log line ties back to one call.
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := firstNonEmpty(
			c.GetHeader(global.RequestID.String()),
			c.GetHeader(global.XRequestID.String()),
			xid.New().String(),
		)
		correlationID := firstNonEmpty(
			c.GetHeader(global.CorrelationID.String()),
			xid.New().String(),
		)

		ctx := global.WithRequestID(c.Request.Context(), requestID)
		ctx = global.WithCorrelationID(ctx, correlationID)
		c.Request = c.Request.WithContext(ctx)

		c.Set(global.RequestID.String(), requestID)
		c.Set(global.CorrelationID.String(), correlationID)
		c.Header(global.XRequestID.String(), requestID)

		c.Next()
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
