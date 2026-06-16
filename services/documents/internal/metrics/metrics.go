package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds business-level Prometheus metrics for the documents service.
type Metrics struct {
	DocumentsTotal       prometheus.Gauge
	DocumentOperations   *prometheus.CounterVec
	DocumentContentBytes prometheus.Histogram
	CoverUploads         *prometheus.CounterVec
}

// New creates documents business metrics registered on reg.
// If reg is nil, prometheus.DefaultRegisterer is used.
func New(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	factory := promauto.With(reg)

	return &Metrics{
		DocumentsTotal: factory.NewGauge(prometheus.GaugeOpts{
			Name: "documents_total",
			Help: "Current total number of documents",
		}),

		DocumentOperations: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "documents_operations_total",
			Help: "Total number of document operations by type and result",
		}, []string{"operation", "status"}),

		DocumentContentBytes: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "documents_content_bytes",
			Help:    "Size in bytes of document content saved on create or update",
			Buckets: prometheus.ExponentialBuckets(1024, 4, 8),
		}),

		CoverUploads: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "documents_cover_uploads_total",
			Help: "Total number of document cover uploads by result",
		}, []string{"status"}),
	}
}
