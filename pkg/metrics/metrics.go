package metrics

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kaanevranportfolio/Commercium/pkg/config"
)

// Registry holds all metrics collectors
type Registry struct {
	registry *prometheus.Registry
	config   config.MetricsConfig

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Business metrics
	activeUsers     prometheus.Gauge
	totalOrders     *prometheus.CounterVec
	paymentStatus   *prometheus.CounterVec
	inventoryLevels *prometheus.GaugeVec

	// System metrics
	goRoutines   prometheus.Gauge
	memoryUsage  prometheus.Gauge
	cpuUsage     prometheus.Gauge
	dbConnections *prometheus.GaugeVec
}

// NewRegistry creates a new metrics registry
func NewRegistry(cfg config.MetricsConfig, serviceName string) (*Registry, error) {
	if !cfg.Enabled {
		return &Registry{config: cfg}, nil
	}

	registry := prometheus.NewRegistry()

	// HTTP metrics
	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code", "service"},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "service"},
	)

	httpRequestSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint", "service"},
	)

	httpResponseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint", "service"},
	)

	// Business metrics
	activeUsers := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "active_users",
			Help:      "Number of currently active users",
		},
	)

	totalOrders := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "orders_total",
			Help:      "Total number of orders",
		},
		[]string{"status", "service"},
	)

	paymentStatus := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "payments_total",
			Help:      "Total number of payments by status",
		},
		[]string{"status", "method", "service"},
	)

	inventoryLevels := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "inventory_level",
			Help:      "Current inventory levels",
		},
		[]string{"product_id", "warehouse", "service"},
	)

	// System metrics
	goRoutines := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "goroutines",
			Help:      "Number of goroutines",
		},
	)

	memoryUsage := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "memory_usage_bytes",
			Help:      "Memory usage in bytes",
		},
	)

	cpuUsage := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "cpu_usage_percent",
			Help:      "CPU usage percentage",
		},
	)

	dbConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "database_connections",
			Help:      "Number of database connections",
		},
		[]string{"database", "state", "service"},
	)

	// Register all metrics
	collectors := []prometheus.Collector{
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestSize,
		httpResponseSize,
		activeUsers,
		totalOrders,
		paymentStatus,
		inventoryLevels,
		goRoutines,
		memoryUsage,
		cpuUsage,
		dbConnections,
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
	}

	// Add Go runtime metrics
	registry.MustRegister(prometheus.NewGoCollector())

	return &Registry{
		registry:            registry,
		config:              cfg,
		httpRequestsTotal:   httpRequestsTotal,
		httpRequestDuration: httpRequestDuration,
		httpRequestSize:     httpRequestSize,
		httpResponseSize:    httpResponseSize,
		activeUsers:         activeUsers,
		totalOrders:         totalOrders,
		paymentStatus:       paymentStatus,
		inventoryLevels:     inventoryLevels,
		goRoutines:          goRoutines,
		memoryUsage:         memoryUsage,
		cpuUsage:            cpuUsage,
		dbConnections:       dbConnections,
	}, nil
}

// Handler returns the HTTP handler for metrics endpoint
func (r *Registry) Handler() http.Handler {
	if !r.config.Enabled {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

// HTTPMiddleware returns Gin middleware for HTTP metrics collection
func (r *Registry) HTTPMiddleware(serviceName string) gin.HandlerFunc {
	if !r.config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		statusCode := string(rune(c.Writer.Status()))

		r.httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			statusCode,
			serviceName,
		).Inc()

		r.httpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			serviceName,
		).Observe(duration)

		if c.Request.ContentLength > 0 {
			r.httpRequestSize.WithLabelValues(
				c.Request.Method,
				c.FullPath(),
				serviceName,
			).Observe(float64(c.Request.ContentLength))
		}

		r.httpResponseSize.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			serviceName,
		).Observe(float64(c.Writer.Size()))
	}
}

// Business metric methods
func (r *Registry) IncActiveUsers() {
	if r.config.Enabled {
		r.activeUsers.Inc()
	}
}

func (r *Registry) DecActiveUsers() {
	if r.config.Enabled {
		r.activeUsers.Dec()
	}
}

func (r *Registry) IncOrdersTotal(status, serviceName string) {
	if r.config.Enabled {
		r.totalOrders.WithLabelValues(status, serviceName).Inc()
	}
}

func (r *Registry) IncPaymentsTotal(status, method, serviceName string) {
	if r.config.Enabled {
		r.paymentStatus.WithLabelValues(status, method, serviceName).Inc()
	}
}

func (r *Registry) SetInventoryLevel(productID, warehouse, serviceName string, level float64) {
	if r.config.Enabled {
		r.inventoryLevels.WithLabelValues(productID, warehouse, serviceName).Set(level)
	}
}

// System metric methods
func (r *Registry) SetGoRoutines(count float64) {
	if r.config.Enabled {
		r.goRoutines.Set(count)
	}
}

func (r *Registry) SetMemoryUsage(bytes float64) {
	if r.config.Enabled {
		r.memoryUsage.Set(bytes)
	}
}

func (r *Registry) SetCPUUsage(percent float64) {
	if r.config.Enabled {
		r.cpuUsage.Set(percent)
	}
}

func (r *Registry) SetDBConnections(database, state, serviceName string, count float64) {
	if r.config.Enabled {
		r.dbConnections.WithLabelValues(database, state, serviceName).Set(count)
	}
}
