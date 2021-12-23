package event

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	methodCalls *prometheus.CounterVec
}

func initMetrics() metrics {
	const ns, sub = "groove", "event"
	return metrics{
		methodCalls: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "method_calls_total",
			Help:      "Total number of method calls",
		}, []string{"method"}),
	}
}

/* To get the method caller name:
(adds a little runtime overhead on each call, look for a way of pre-defining and
associating each method with its name instead of calculating it every time)

var mp = map[uintptr]string{}
fpcs := make([]uintptr, 1)
// Skip 2 levels to get the caller
if n := runtime.Callers(2, fpcs); n == 0 {
	return ""
}

fpc := fpcs[0] - 1
if v, ok := mp[fpc]; ok {
	return v
}

caller := runtime.FuncForPC(fpc)
if caller == nil {
	return ""
}

fullName := caller.Name()
lastIdx := strings.LastIndexByte(fullName, '.')

name := fullName[lastIdx+1:]
mp[fpc] = name

return name
*/
func (m metrics) incMethodCalls(method string) {
	m.methodCalls.With(prometheus.Labels{"method": method}).Inc()
}
