package notification

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	methodCalls *prometheus.CounterVec
	sent        prometheus.Counter
	fail        prometheus.Counter
}

func initMetrics() metrics {
	const ns, sub = "groove", "notification"
	return metrics{
		methodCalls: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "method_calls_total",
			Help:      "Total number of method calls",
		}, []string{"method"}),
		sent: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "sent_success",
			Help:      "Total number of successfully sent notifications",
		}),
		fail: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "sent_fail",
			Help:      "Total number of notifications failed to sent",
		}),
	}
}

func (m metrics) incMethodCalls(method string) {
	m.methodCalls.With(prometheus.Labels{"method": method}).Inc()
}
