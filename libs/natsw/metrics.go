package natsw

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds Prometheus metrics for NATS request handling.
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

// NewMetrics creates NATS request metrics registered on reg.
// If reg is nil, prometheus.DefaultRegisterer is used.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	factory := promauto.With(reg)

	return &Metrics{
		RequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nats_requests_total",
			Help: "The total number of processed NATS requests",
		}, []string{"subject", "status"}),

		RequestDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nats_request_duration_seconds",
			Help:    "A histogram of NATS handler durations",
			Buckets: prometheus.DefBuckets,
		}, []string{"subject", "status"}),
	}
}

// Middleware records request count and duration metrics for each handled message.
func (m *Metrics) Middleware(next HandlerFunc) HandlerFunc {
	return func(msg *Message) error {
		start := time.Now()
		err := next(msg)
		duration := time.Since(start).Seconds()

		status := "success"
		if err != nil {
			status = "error"
		}

		m.RequestsTotal.WithLabelValues(msg.Subject, status).Inc()
		m.RequestDuration.WithLabelValues(msg.Subject, status).Observe(duration)

		return err
	}
}
