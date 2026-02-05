package middleware

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/artmexbet/raibecas/libs/natsw"
)

type Metrics struct {
	OpsProcessed    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	UsersTotal      prometheus.Gauge
	UserRegistered  prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	factory := promauto.With(reg)

	m := &Metrics{
		OpsProcessed: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nats_requests_total",
			Help: "The total number of processed NATS requests",
		}, []string{"subject", "status"}),

		RequestDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nats_request_duration_seconds",
			Help:    "A histogram of the request duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"subject", "status"}),

		UsersTotal: factory.NewGauge(prometheus.GaugeOpts{
			Name: "users_total_count",
			Help: "Total number of users in database",
		}),

		UserRegistered: factory.NewCounter(prometheus.CounterOpts{
			Name: "users_registered_total",
			Help: "Total number of newly registered users since app start",
		}),
	}

	return m
}

func (m *Metrics) Middleware(next natsw.HandlerFunc) natsw.HandlerFunc {
	return func(msg *natsw.Message) error {
		start := time.Now()
		err := next(msg)
		duration := time.Since(start).Seconds()

		status := "success"
		if err != nil {
			status = "error"
		}

		m.OpsProcessed.WithLabelValues(msg.Subject, status).Inc()
		m.RequestDuration.WithLabelValues(msg.Subject, status).Observe(duration)

		return err
	}
}

// IncRegisteredUsers increments the counter of registered users.
// Used to satisfy service.Metrics interface.
func (m *Metrics) IncRegisteredUsers() {
	m.UserRegistered.Inc()
}
