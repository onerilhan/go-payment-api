package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// MetricsConfig middleware ayarları
type MetricsConfig struct {
	EnableMemoryTracking  bool
	EnableResponseTime    bool
	EnableRequestCount    bool
	EnableConcurrency     bool
	EnableStatusCodeCount bool

	SlowRequestThreshold time.Duration // Yavaş istek eşiği
	MemoryAlertThreshold uint64        // Bellek kullanım eşiği (bytes)
	MaxStoredResponse    int           // Kaç adet response time saklanacak
	MemoryCheckInterval  time.Duration // Bellek kontrol sıklığı
}

// Varsayılan config
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		EnableMemoryTracking:  true,
		EnableResponseTime:    true,
		EnableRequestCount:    true,
		EnableConcurrency:     true,
		EnableStatusCodeCount: true,
		SlowRequestThreshold:  2 * time.Second,
		MemoryAlertThreshold:  100 * 1024 * 1024, // 100MB
		MaxStoredResponse:     100,
		MemoryCheckInterval:   30 * time.Second,
	}
}

// Metrics verileri
type Metrics struct {
	mutex               sync.RWMutex
	TotalRequests       int64
	ActiveRequests      int64
	ResponseTimes       map[string][]time.Duration
	StatusCodeCounts    map[int]int64
	EndpointCounts      map[string]int64
	SlowRequests        int64
	MemoryUsage         uint64
	LastMemoryCheck     time.Time
	AverageResponseTime time.Duration
}

// Snapshot formatı (JSON response)
type MetricsSnapshot struct {
	TotalRequests       int64                       `json:"total_requests"`
	ActiveRequests      int64                       `json:"active_requests"`
	SlowRequests        int64                       `json:"slow_requests"`
	MemoryUsage         uint64                      `json:"memory_usage_bytes"`
	AverageResponseTime time.Duration               `json:"average_response_time"`
	StatusCodeCounts    map[int]int64               `json:"status_code_counts"`
	EndpointCounts      map[string]int64            `json:"endpoint_counts"`
	ResponseTimeSummary map[string]ResponseTimeStat `json:"response_time_summary"`
	LastUpdated         time.Time                   `json:"last_updated"`
}

// ResponseTimeStat ek olarak percentil veriyor
type ResponseTimeStat struct {
	Count   int           `json:"count"`
	Average time.Duration `json:"average"`
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
}

// Custom response writer (status code yakalamak için)
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (mrw *metricsResponseWriter) WriteHeader(code int) {
	mrw.statusCode = code
	mrw.ResponseWriter.WriteHeader(code)
}

// NewMetricsMiddleware middleware + handler döner
func NewMetricsMiddleware(ctx context.Context, config *MetricsConfig) (func(http.Handler) http.Handler, http.HandlerFunc) {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	metrics := &Metrics{
		ResponseTimes:    make(map[string][]time.Duration),
		StatusCodeCounts: make(map[int]int64),
		EndpointCounts:   make(map[string]int64),
	}

	// Memory monitor başlat
	if config.EnableMemoryTracking {
		go memoryMonitor(ctx, metrics, config)
	}

	// Middleware
	middlewareFunc := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			if config.EnableRequestCount {
				metrics.mutex.Lock()
				metrics.TotalRequests++
				if config.EnableConcurrency {
					metrics.ActiveRequests++
				}
				metrics.EndpointCounts[r.URL.Path]++
				metrics.mutex.Unlock()
			}

			wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: 200}
			next.ServeHTTP(wrapped, r)

			elapsed := time.Since(start)

			metrics.mutex.Lock()
			if config.EnableConcurrency {
				metrics.ActiveRequests--
			}

			if config.EnableResponseTime {
				if metrics.ResponseTimes[r.URL.Path] == nil {
					metrics.ResponseTimes[r.URL.Path] = []time.Duration{}
				}
				rtList := append(metrics.ResponseTimes[r.URL.Path], elapsed)
				if len(rtList) > config.MaxStoredResponse {
					rtList = rtList[len(rtList)-config.MaxStoredResponse:]
				}
				metrics.ResponseTimes[r.URL.Path] = rtList
				updateAverage(metrics)
			}

			if config.EnableStatusCodeCount {
				metrics.StatusCodeCounts[wrapped.statusCode]++
			}

			if elapsed > config.SlowRequestThreshold {
				metrics.SlowRequests++
				log.Warn().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Dur("response_time", elapsed).
					Msg("Slow request detected")
			}

			metrics.mutex.Unlock()

			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Dur("response_time", elapsed).
				Int("status_code", wrapped.statusCode).
				Int64("active_requests", metrics.ActiveRequests).
				Msg("Request metrics")
		})
	}

	// Handler (JSON output)
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		snapshot := getSnapshot(metrics)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snapshot)
	}

	return middlewareFunc, handlerFunc
}

// Bellek monitor
func memoryMonitor(ctx context.Context, m *Metrics, config *MetricsConfig) {
	ticker := time.NewTicker(config.MemoryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Memory monitor stopped")
			return
		case <-ticker.C:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)

			m.mutex.Lock()
			m.MemoryUsage = mem.Alloc
			m.LastMemoryCheck = time.Now()
			m.mutex.Unlock()

			if mem.Alloc > config.MemoryAlertThreshold {
				log.Warn().
					Uint64("current_memory", mem.Alloc).
					Msg("High memory usage detected")
			}
		}
	}
}

// Ortalama response time güncelle
func updateAverage(m *Metrics) {
	var total time.Duration
	var count int64
	for _, times := range m.ResponseTimes {
		for _, t := range times {
			total += t
			count++
		}
	}
	if count > 0 {
		m.AverageResponseTime = total / time.Duration(count)
	}
}

// Snapshot oluştur
func getSnapshot(m *Metrics) *MetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	summary := make(map[string]ResponseTimeStat)
	for path, times := range m.ResponseTimes {
		if len(times) > 0 {
			stat := ResponseTimeStat{
				Count:   len(times),
				Average: avg(times),
				Min:     min(times),
				Max:     max(times),
				P95:     percentile(times, 95),
				P99:     percentile(times, 99),
			}
			summary[path] = stat
		}
	}

	return &MetricsSnapshot{
		TotalRequests:       m.TotalRequests,
		ActiveRequests:      m.ActiveRequests,
		SlowRequests:        m.SlowRequests,
		MemoryUsage:         m.MemoryUsage,
		AverageResponseTime: m.AverageResponseTime,
		StatusCodeCounts:    copyMap(m.StatusCodeCounts),
		EndpointCounts:      copyMap(m.EndpointCounts),
		ResponseTimeSummary: summary,
		LastUpdated:         time.Now(),
	}
}

// Yardımcı fonksiyonlar
func avg(times []time.Duration) time.Duration {
	var total time.Duration
	for _, t := range times {
		total += t
	}
	return total / time.Duration(len(times))
}

func min(times []time.Duration) time.Duration {
	min := times[0]
	for _, t := range times {
		if t < min {
			min = t
		}
	}
	return min
}

func max(times []time.Duration) time.Duration {
	max := times[0]
	for _, t := range times {
		if t > max {
			max = t
		}
	}
	return max
}

func percentile(times []time.Duration, p int) time.Duration {
	if len(times) == 0 {
		return 0
	}
	sorted := append([]time.Duration{}, times...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted))*float64(p)/100.0 + 0.5)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func copyMap[K comparable, V any](original map[K]V) map[K]V {
	out := make(map[K]V)
	for k, v := range original {
		out[k] = v
	}
	return out
}
