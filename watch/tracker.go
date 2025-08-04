package watch

import (
	"sort"
	"sync"
)

const maxHistorySize = 100

type ThresholdTracker struct {
	mu           sync.RWMutex
	history      map[string][]float64
	minEvents    int
	multiplier   float64
	fallbackBase float64
}

func NewThresholdTracker(minEvents int, multiplier float64) *ThresholdTracker {
	return &ThresholdTracker{
		history:      make(map[string][]float64),
		minEvents:    minEvents,
		multiplier:   multiplier,
		fallbackBase: 0,
	}
}

func (tt *ThresholdTracker) Add(camera string, percent float64) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	h := tt.history[camera]
	l := len(h)

	if l == 0 {
		h = append(h, tt.fallbackBase)
	}

	if l >= maxHistorySize+1 {
		h = h[1:]
	}

	h = append(h, percent)
	tt.history[camera] = h
}

func (tt *ThresholdTracker) Get(camera string, fallback float64) float64 {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	h, ok := tt.history[camera]
	l := len(h)

	if !ok || l == 0 {
		tt.fallbackBase = fallback
		return fallback
	}

	if l < tt.minEvents {
		return fallback
	}

	copied := make([]float64, l)
	copy(copied, h)
	sort.Float64s(copied)

	percentile95 := copied[int(float64(len(copied))*0.95)]

	var sum float64
	for _, v := range copied {
		sum += v
	}
	avg := sum / float64(len(copied))

	return maxVal(avg*tt.multiplier, percentile95)
}

func maxVal(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
