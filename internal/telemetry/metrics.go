package telemetry

import (
	"sync/atomic"
)

// MetricsSnapshot represents a point-in-time snapshot of telemetry metrics.
type MetricsSnapshot struct {
	TotalRequests        uint64  `json:"total_requests"`
	RepairedRequests     uint64  `json:"repaired_requests"`
	ValidPassThrough     uint64  `json:"valid_pass_through"`
	FailedRepairs        uint64  `json:"failed_repairs"`
	SuccessRate          float64 `json:"success_rate"`
	TokensSavedEstimated uint64  `json:"tokens_saved_estimated"`
	AvgLatencyUs         float64 `json:"avg_latency_us"`
	TruncationFixed      uint64  `json:"truncation_fixed"`
	MarkdownStripped     uint64  `json:"markdown_stripped"`
	QuotesFixed          uint64  `json:"quotes_fixed"`
	TrailingCommasFixed  uint64  `json:"trailing_commas_fixed"`
	DanglingKeysFixed    uint64  `json:"dangling_keys_fixed"`
	TypeCoercions        uint64  `json:"type_coercions"`
}

// Metrics tracks cumulative counters using atomic operations.
type Metrics struct {
	totalRequests        atomic.Uint64
	repairedRequests     atomic.Uint64
	validPassThrough     atomic.Uint64
	failedRepairs        atomic.Uint64
	tokensSavedEstimated atomic.Uint64
	totalLatencyUs       atomic.Uint64

	truncationFixed     atomic.Uint64
	markdownStripped    atomic.Uint64
	quotesFixed         atomic.Uint64
	trailingCommasFixed atomic.Uint64
	danglingKeysFixed   atomic.Uint64
	typeCoercions       atomic.Uint64
}

// NewMetrics initializes atomic counters.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Record updates metrics based on an executed repair.
func (m *Metrics) Record(repaired bool, valid bool, latencyUs int64, tokensSaved int64,
	truncation, markdown, quotes, trailingCommas, danglingKeys, coercions bool) {

	m.totalRequests.Add(1)
	if latencyUs > 0 {
		m.totalLatencyUs.Add(uint64(latencyUs))
	}

	if repaired {
		m.repairedRequests.Add(1)
		if tokensSaved > 0 {
			m.tokensSavedEstimated.Add(uint64(tokensSaved))
		}
	} else if valid {
		m.validPassThrough.Add(1)
	}

	if !valid {
		m.failedRepairs.Add(1)
	}

	if truncation {
		m.truncationFixed.Add(1)
	}
	if markdown {
		m.markdownStripped.Add(1)
	}
	if quotes {
		m.quotesFixed.Add(1)
	}
	if trailingCommas {
		m.trailingCommasFixed.Add(1)
	}
	if danglingKeys {
		m.danglingKeysFixed.Add(1)
	}
	if coercions {
		m.typeCoercions.Add(1)
	}
}

// Snapshot returns the current snapshot of metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	total := m.totalRequests.Load()
	repaired := m.repairedRequests.Load()
	passThrough := m.validPassThrough.Load()
	failed := m.failedRepairs.Load()
	totalLat := m.totalLatencyUs.Load()

	var successRate float64 = 100.0
	if total > 0 {
		successRate = float64(repaired+passThrough) / float64(total) * 100.0
	}

	var avgLat float64 = 0.0
	if total > 0 {
		avgLat = float64(totalLat) / float64(total)
	}

	return MetricsSnapshot{
		TotalRequests:        total,
		RepairedRequests:     repaired,
		ValidPassThrough:     passThrough,
		FailedRepairs:        failed,
		SuccessRate:          successRate,
		TokensSavedEstimated: m.tokensSavedEstimated.Load(),
		AvgLatencyUs:         avgLat,
		TruncationFixed:      m.truncationFixed.Load(),
		MarkdownStripped:     m.markdownStripped.Load(),
		QuotesFixed:          m.quotesFixed.Load(),
		TrailingCommasFixed:  m.trailingCommasFixed.Load(),
		DanglingKeysFixed:    m.danglingKeysFixed.Load(),
		TypeCoercions:        m.typeCoercions.Load(),
	}
}
