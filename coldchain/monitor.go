package coldchain

import (
	"context"
	"fmt"
	"sync"
)

// Monitor keeps a bounded sliding history of readings per device and counts
// excursions. Every read path returns independent copies so callers cannot
// race with the writer that appends into the ring buffer.
type Monitor struct {
	mu         sync.RWMutex
	history    map[string][]Reading
	window     int
	policy     *Policy
	excursions map[string]int
}

func NewMonitor(window int, policy *Policy) *Monitor {
	if window < 1 {
		window = 60
	}
	return &Monitor{
		history:    map[string][]Reading{},
		window:     window,
		policy:     policy,
		excursions: map[string]int{},
	}
}

// ProcessReading records the reading, trims the ring buffer to the window and
// returns an alert when the temperature leaves the normal band. The excursion
// counter is incremented exactly once per out-of-band reading.
func (m *Monitor) ProcessReading(ctx context.Context, r Reading) ([]Alert, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ring := m.history[r.DeviceID]
	if len(ring) < m.window {
		ring = append(ring, r)
	} else {
		copy(ring, ring[1:])
		ring[len(ring)-1] = r
	}
	m.history[r.DeviceID] = ring
	level := "normal"
	if m.policy != nil {
		level = m.policy.Level(r.TempC)
	}
	if level == "warning" || level == "critical" {
		m.excursions[r.DeviceID]++
		return []Alert{{DeviceID: r.DeviceID, Level: level, Message: fmt.Sprintf("temp %.1fC", r.TempC)}}, nil
	}
	return nil, nil
}

// Recent returns the most recent n readings for a device as an independent
// copy.
func (m *Monitor) Recent(ctx context.Context, deviceID string, n int) ([]Reading, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	ring := m.history[deviceID]
	if n < 1 || n > len(ring) {
		n = len(ring)
	}
	if n == 0 {
		return []Reading{}, nil
	}
	return ring[len(ring)-n:], nil
}

func (m *Monitor) Excursions(deviceID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.excursions[deviceID]
}
