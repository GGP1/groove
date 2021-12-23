package user

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	methodCalls *prometheus.CounterVec
}

func initMetrics() metrics {
	const ns, sub = "groove", "user"
	return metrics{
		methodCalls: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "method_calls_total",
			Help:      "Total number of method calls",
		}, []string{"method"}),
	}
}

func (m metrics) incMethodCalls(method string) {
	m.methodCalls.With(prometheus.Labels{"method": method}).Inc()
}
