package postgres

import (
	"database/sql"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	open              prometheus.Gauge
	maxOpen           prometheus.Gauge
	inUse             prometheus.Gauge
	idle              prometheus.Gauge
	wait              prometheus.Gauge
	waitDuration      prometheus.Gauge
	maxIdleClosed     prometheus.Gauge
	maxIdleTimeClosed prometheus.Gauge
	maxLifetimeClosed prometheus.Gauge
}

func runMetrics(db *sql.DB, config config.Postgres) {
	ns, sub := "groove", "postgres"
	m := metrics{
		open: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "open_connections_total",
			Help:      "Total number of concurrent active connections",
		}),
		maxOpen: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "max_open_connections_total",
			Help:      "Maximum number of concurrent active connections",
		}),
		inUse: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "in_use_connections_total",
			Help:      "Total number of connections in use",
		}),
		idle: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "idle_connections_total",
			Help:      "Total number of idle connections",
		}),
		wait: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "wait_total",
			Help:      "Total number of connections waited for",
		}),
		waitDuration: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "wait_duration_total",
			Help:      "Total time blocked waiting for a new connection",
		}),
		maxIdleClosed: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "max_idle_closed_total",
			Help:      "Total number of connections closed due to SetMaxIdleConns",
		}),
		maxIdleTimeClosed: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "max_idle_time_closed_total",
			Help:      "Total number of connections closed due to SetConnMaxIdleTime",
		}),
		maxLifetimeClosed: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "max_lifetime_closed_total",
			Help:      "Total number of connections closed due to SetConnMaxLifetime",
		}),
	}

	if config.MetricsRate != 0 {
		go m.Run(db, config)
	}
}

func (m metrics) Run(db *sql.DB, config config.Postgres) {
	for {
		time.Sleep(config.MetricsRate * time.Second)
		stats := db.Stats()
		m.open.Set(float64(stats.OpenConnections))
		m.maxOpen.Set(float64(stats.MaxOpenConnections))
		m.inUse.Set(float64(stats.InUse))
		m.idle.Set(float64(stats.Idle))
		m.wait.Set(float64(stats.WaitCount))
		m.waitDuration.Set(float64(stats.WaitDuration))
		m.maxIdleClosed.Set(float64(stats.MaxIdleClosed))
		m.maxIdleTimeClosed.Set(float64(stats.MaxIdleTimeClosed))
		m.maxLifetimeClosed.Set(float64(stats.MaxLifetimeClosed))
	}
}
