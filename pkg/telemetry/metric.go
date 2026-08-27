package telemetry

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metrics holds the collectors exposed on /metrics.
type metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

func newMetrics(serviceName string) *metrics {
	labels := prometheus.Labels{"service": serviceName}

	return &metrics{
		requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests handled, by route and status.",
			ConstLabels: labels,
		}, []string{"method", "route", "status"}),

		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request latency in seconds, by route.",
			ConstLabels: labels,
			Buckets:     prometheus.DefBuckets,
		}, []string{"method", "route"}),

		inFlight: promauto.NewGauge(prometheus.GaugeOpts{
			Name:        "http_requests_in_flight",
			Help:        "Number of HTTP requests currently being served.",
			ConstLabels: labels,
		}),
	}
}

// middleware records count, latency and in-flight per request. Routes use the
// matched template, so path params cannot explode cardinality.
func (m *metrics) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		m.requests.WithLabelValues(c.Request.Method, route, strconv.Itoa(c.Writer.Status())).Inc()
		m.duration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}
