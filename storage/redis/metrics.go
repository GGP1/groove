package redis

import (
	"time"

	"github.com/GGP1/groove/config"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	lastRun    prometheus.Gauge
	hits       prometheus.Gauge
	misses     prometheus.Gauge
	timeouts   prometheus.Gauge
	openConns  prometheus.Gauge
	idleConns  prometheus.Gauge
	staleConns prometheus.Gauge
}

func runMetrics(rdb *redis.Client, config config.Redis) {
	ns, sub := "groove", "redis"
	m := metrics{
		lastRun: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "last_run",
			Help:      "Last gather date in Unix seconds",
		}),
		hits: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "hits_total",
			Help:      "Total number of times a free connection was found in the pool",
		}),
		misses: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "misses_total",
			Help:      "Total number of times a free connection was not found in the pool",
		}),
		timeouts: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "timeouts_total",
			Help:      "Total number of times a wait timeout occurred",
		}),
		openConns: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "open_conns_total",
			Help:      "Total number of open connections in the pool",
		}),
		idleConns: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "idle_conns_total",
			Help:      "Total number of idle connections in the pool",
		}),
		staleConns: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "stale_conns_total",
			Help:      "Total number of stale connections removed from the pool",
		}),
	}

	m.Run(rdb, config)
}

func (m metrics) Run(rdb *redis.Client, config config.Redis) {
	go func() {
		for {
			time.Sleep(config.MetricsRate)
			stats := rdb.PoolStats()
			m.lastRun.SetToCurrentTime()
			m.hits.Set(float64(stats.Hits))
			m.misses.Set(float64(stats.Misses))
			m.timeouts.Set(float64(stats.Timeouts))
			m.openConns.Set(float64(stats.TotalConns))
			m.idleConns.Set(float64(stats.IdleConns))
			m.staleConns.Set(float64(stats.StaleConns))
		}
	}()
}
