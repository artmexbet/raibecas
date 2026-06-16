package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds business-level Prometheus metrics for the chat service.
type Metrics struct {
	MessagesTotal         *prometheus.CounterVec
	EmbeddingDuration     prometheus.Histogram
	VectorSearchDuration  prometheus.Histogram
	LLMGenerationDuration prometheus.Histogram
	RetrievedDocuments    prometheus.Histogram
}

// New creates chat business metrics registered on reg.
// If reg is nil, prometheus.DefaultRegisterer is used.
func New(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	factory := promauto.With(reg)

	return &Metrics{
		MessagesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "chat_messages_total",
			Help: "Total number of chat messages processed by result",
		}, []string{"status"}),

		EmbeddingDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "chat_embedding_duration_seconds",
			Help:    "Duration of embedding generation for chat input",
			Buckets: prometheus.DefBuckets,
		}),

		VectorSearchDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "chat_vector_search_duration_seconds",
			Help:    "Duration of vector store retrieval for chat input",
			Buckets: prometheus.DefBuckets,
		}),

		LLMGenerationDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "chat_llm_generation_duration_seconds",
			Help:    "Duration of LLM response generation for chat input",
			Buckets: prometheus.ExponentialBuckets(0.5, 2, 10),
		}),

		RetrievedDocuments: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "chat_retrieved_documents",
			Help:    "Number of documents retrieved from the vector store per chat request",
			Buckets: prometheus.LinearBuckets(0, 1, 11),
		}),
	}
}
