// Package metrics exposes Prometheus instrumentation for the API: HTTP
// latency/error rates, pgx connection-pool health, Redis pool health, and Go
// runtime stats. Scraped at GET /metrics.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	goredis "github.com/redis/go-redis/v9"
)

var (
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by route, method and status.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
	}, []string{"method", "route", "status"})

	httpTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by route, method and status.",
	}, []string{"method", "route", "status"})

	httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "In-flight HTTP requests.",
	})

	// ── Biznes / WS metrikalari ──
	RoomTokensIssued = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "darsly_room_tokens_issued_total",
		Help: "LiveKit token'lar soni (rol bo'yicha).",
	}, []string{"role"})

	WaitingRoomDecisions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "darsly_waitingroom_decisions_total",
		Help: "Kutish xonasi qarorlari (admit/reject).",
	}, []string{"decision"})

	RecordingsStarted = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "darsly_recordings_started_total",
		Help: "Boshlangan yozuvlar soni.",
	})

	NotificationsSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "darsly_notifications_sent_total",
		Help: "Yuborilgan bildirishnomalar soni.",
	})

	// ── Xatolik metrikalari (failure) ──
	LiveKitErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "darsly_livekit_errors_total",
		Help: "LiveKit operatsiya xatolari (op bo'yicha).",
	}, []string{"op"}) // create_room, host_token, egress_start, egress_stop, mute, remove, ...

	EgressResults = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "darsly_egress_results_total",
		Help: "Egress (recording) yakuniy natijalari.",
	}, []string{"result"}) // ready, failed

	WSConnections = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "darsly_ws_connections_total",
		Help: "WebSocket ulanish hodisalari.",
	}, []string{"result"}) // opened, upgrade_failed, closed

	// WSDropped — sekin klient sabab tashlab yuborilgan WS xabarlar (backpressure).
	// O'sib borsa — sekin-klient muammosi (past internet) yoki bufer kichikligi belgisi.
	WSDropped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "darsly_ws_messages_dropped_total",
		Help: "Backpressure sabab tashlangan WebSocket xabarlar soni.",
	})

	// ── Latency histogrammalari ──
	LiveKitOpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "darsly_livekit_op_duration_seconds",
		Help:    "LiveKit operatsiya davomiyligi (op bo'yicha).",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	}, []string{"op"})

	registry = prometheus.NewRegistry()
)

func init() {
	registry.MustRegister(httpDuration, httpTotal, httpInFlight)
	registry.MustRegister(RoomTokensIssued, WaitingRoomDecisions, RecordingsStarted, NotificationsSent)
	registry.MustRegister(LiveKitErrors, EgressResults, WSConnections, WSDropped, LiveKitOpDuration)
	// Go runtime (goroutines, GC pauses, heap) + process (CPU, FDs, RSS).
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

// RegisterWSActive faol WebSocket ulanishlar sonini kuzatuvchi gauge'ni ro'yxatga oladi.
func RegisterWSActive(fn func() float64) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "darsly_ws_active_connections",
		Help: "Shu instansdagi faol WebSocket ulanishlari.",
	}, fn))
}

// RegisterPool wires a pgx pool collector so pool saturation is observable.
func RegisterPool(pool *pgxpool.Pool) {
	if pool != nil {
		registry.MustRegister(&pgxPoolCollector{pool: pool})
	}
}

// RegisterRedis wires a Redis pool collector.
func RegisterRedis(client *goredis.Client) {
	if client != nil {
		registry.MustRegister(&redisPoolCollector{client: client})
	}
}

// Middleware records latency, count and in-flight for every request, labeled
// by the matched route pattern (low cardinality — IDs are not in the label).
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		httpInFlight.Inc()
		c.Next()
		httpInFlight.Dec()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		httpDuration.WithLabelValues(c.Request.Method, route, status).Observe(time.Since(start).Seconds())
		httpTotal.WithLabelValues(c.Request.Method, route, status).Inc()
	}
}

// Handler serves the Prometheus exposition format.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// ─── pgx pool collector ────────────────────────────────────────────────────

type pgxPoolCollector struct{ pool *pgxpool.Pool }

var (
	poolTotal    = prometheus.NewDesc("pgx_pool_conns_total", "Total connections in the pgx pool.", nil, nil)
	poolAcquired = prometheus.NewDesc("pgx_pool_conns_acquired", "Currently acquired (in-use) connections.", nil, nil)
	poolIdle     = prometheus.NewDesc("pgx_pool_conns_idle", "Idle connections.", nil, nil)
	poolMax      = prometheus.NewDesc("pgx_pool_conns_max", "Configured max connections.", nil, nil)
	// EmptyAcquireCount = pool bo'sh bo'lgani uchun kutgan acquire'lar (haqiqiy "wait").
	poolEmptyCnt = prometheus.NewDesc("pgx_pool_empty_acquire_total", "Acquires that had to wait because the pool was empty.", nil, nil)
	// AcquireDuration = barcha acquire'larга ketgan JAMI vaqt (faqat "wait" emas).
	poolAcquireSecs = prometheus.NewDesc("pgx_pool_acquire_duration_seconds_total", "Total time spent acquiring connections.", nil, nil)
)

func (c *pgxPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- poolTotal
	ch <- poolAcquired
	ch <- poolIdle
	ch <- poolMax
	ch <- poolEmptyCnt
	ch <- poolAcquireSecs
}

func (c *pgxPoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(poolTotal, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(poolAcquired, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(poolIdle, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(poolMax, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(poolEmptyCnt, prometheus.CounterValue, float64(s.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(poolAcquireSecs, prometheus.CounterValue, s.AcquireDuration().Seconds())
}

// ─── Redis pool collector ──────────────────────────────────────────────────

type redisPoolCollector struct{ client *goredis.Client }

var (
	redisTotal   = prometheus.NewDesc("redis_pool_conns_total", "Total connections in the Redis pool.", nil, nil)
	redisIdle    = prometheus.NewDesc("redis_pool_conns_idle", "Idle Redis connections.", nil, nil)
	redisStale   = prometheus.NewDesc("redis_pool_conns_stale_total", "Stale connections removed.", nil, nil)
	redisTimeout = prometheus.NewDesc("redis_pool_timeouts_total", "Connection pool timeouts.", nil, nil)
	redisHits    = prometheus.NewDesc("redis_pool_hits_total", "Free connection found in the pool.", nil, nil)
	redisMisses  = prometheus.NewDesc("redis_pool_misses_total", "Free connection not found in the pool.", nil, nil)
)

func (c *redisPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- redisTotal
	ch <- redisIdle
	ch <- redisStale
	ch <- redisTimeout
	ch <- redisHits
	ch <- redisMisses
}

func (c *redisPoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.client.PoolStats()
	ch <- prometheus.MustNewConstMetric(redisTotal, prometheus.GaugeValue, float64(s.TotalConns))
	ch <- prometheus.MustNewConstMetric(redisIdle, prometheus.GaugeValue, float64(s.IdleConns))
	ch <- prometheus.MustNewConstMetric(redisStale, prometheus.CounterValue, float64(s.StaleConns))
	ch <- prometheus.MustNewConstMetric(redisTimeout, prometheus.CounterValue, float64(s.Timeouts))
	ch <- prometheus.MustNewConstMetric(redisHits, prometheus.CounterValue, float64(s.Hits))
	ch <- prometheus.MustNewConstMetric(redisMisses, prometheus.CounterValue, float64(s.Misses))
}
