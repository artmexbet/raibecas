package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds business-level Prometheus metrics for the auth service.
type Metrics struct {
	LoginAttempts        *prometheus.CounterVec
	RegistrationRequests prometheus.Counter
	TokenRefreshes       *prometheus.CounterVec
	PasswordChanges      prometheus.Counter
	Logouts              prometheus.Counter
}

// New creates auth business metrics registered on reg.
// If reg is nil, prometheus.DefaultRegisterer is used.
func New(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	factory := promauto.With(reg)

	return &Metrics{
		LoginAttempts: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "auth_login_attempts_total",
			Help: "Total number of login attempts by result",
		}, []string{"status"}),

		RegistrationRequests: factory.NewCounter(prometheus.CounterOpts{
			Name: "auth_registration_requests_total",
			Help: "Total number of registration requests submitted",
		}),

		TokenRefreshes: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "auth_token_refresh_total",
			Help: "Total number of token refresh attempts by result",
		}, []string{"status"}),

		PasswordChanges: factory.NewCounter(prometheus.CounterOpts{
			Name: "auth_password_changes_total",
			Help: "Total number of successful password changes",
		}),

		Logouts: factory.NewCounter(prometheus.CounterOpts{
			Name: "auth_logouts_total",
			Help: "Total number of logout operations",
		}),
	}
}
