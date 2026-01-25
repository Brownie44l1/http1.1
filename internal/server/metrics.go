package server

import (
	"sync/atomic"
	"time"
)

// ✅ Issue #16: Metrics and Observability

// Metrics holds server runtime metrics
type Metrics struct {
	RequestsTotal     atomic.Int64
	ActiveConnections atomic.Int64
	ErrorsTotal       atomic.Int64
	Errors4xx         atomic.Int64
	Errors5xx         atomic.Int64
	
	// Latency tracking (simplified - use histogram in production)
	TotalLatencyNs atomic.Int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordRequest records a completed request
func (m *Metrics) RecordRequest(statusCode int, duration time.Duration) {
	m.RequestsTotal.Add(1)
	m.TotalLatencyNs.Add(duration.Nanoseconds())
	
	if statusCode >= 400 && statusCode < 500 {
		m.Errors4xx.Add(1)
	} else if statusCode >= 500 {
		m.Errors5xx.Add(1)
		m.ErrorsTotal.Add(1)
	}
}

// AverageLatency returns average request latency
func (m *Metrics) AverageLatency() time.Duration {
	totalReqs := m.RequestsTotal.Load()
	if totalReqs == 0 {
		return 0
	}
	
	avgNs := m.TotalLatencyNs.Load() / totalReqs
	return time.Duration(avgNs)
}

// Snapshot returns a snapshot of current metrics
type MetricsSnapshot struct {
	RequestsTotal     int64
	ActiveConnections int64
	ErrorsTotal       int64
	Errors4xx         int64
	Errors5xx         int64
	AverageLatency    time.Duration
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		RequestsTotal:     m.RequestsTotal.Load(),
		ActiveConnections: m.ActiveConnections.Load(),
		ErrorsTotal:       m.ErrorsTotal.Load(),
		Errors4xx:         m.Errors4xx.Load(),
		Errors5xx:         m.Errors5xx.Load(),
		AverageLatency:    m.AverageLatency(),
	}
}