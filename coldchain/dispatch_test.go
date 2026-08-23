package coldchain

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

func policyForDispatchTest() *Policy {
	return &Policy{WarningHigh: 8, WarningLow: 2, CriticalHigh: 10, CriticalLow: 0}
}

// TestDispatchConcurrentSubmission verifies every submitted reading is drained
// by the worker pool and processed exactly once.
func TestDispatchConcurrentSubmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor := NewMonitor(64, policyForDispatchTest())
	notifier := NewNotifier()
	dispatcher := NewDispatcher(monitor, notifier, 2)
	dispatcher.Start(ctx)

	readings := make([]Reading, 0, 64)
	for i := 0; i < 64; i++ {
		readings = append(readings, Reading{DeviceID: "dev-a", TempC: 12.0, At: "t0"})
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = dispatcher.Dispatch(ctx, readings)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatch hung: submissions were not drained")
	}
	deadline := time.Now().Add(3 * time.Second)
	for monitor.Excursions("dev-a") != 64 && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if got := monitor.Excursions("dev-a"); got != 64 {
		t.Fatalf("expected 64 processed readings, got %d excursions", got)
	}
	dispatcher.Stop()
}

// TestWorkerPoolShutsDownOnCancel verifies that cancelling the run context
// makes every worker exit; a pool that parks workers on the job channel leaks
// goroutines.
func TestWorkerPoolShutsDownOnCancel(t *testing.T) {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	monitor := NewMonitor(64, policyForDispatchTest())
	notifier := NewNotifier()
	dispatcher := NewDispatcher(monitor, notifier, 4)
	dispatcher.Start(ctx)
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runtime.Gosched()
		if runtime.NumGoroutine() <= before+1 {
			break
		}
	}
	dispatcher.Stop()
	if got := runtime.NumGoroutine(); got > before+1 {
		t.Fatalf("worker goroutines did not stop after cancel: before=%d after=%d", before, got)
	}
}

// TestRecentReadingsSnapshotIsolation drives the reader and the writer
// concurrently: Recent must never expose the internal ring buffer to the
// writer.
func TestRecentReadingsSnapshotIsolation(t *testing.T) {
	ctx := context.Background()
	monitor := NewMonitor(8, policyForDispatchTest())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			_, _ = monitor.ProcessReading(ctx, Reading{DeviceID: "dev-d", TempC: 12.0})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			readings, _ := monitor.Recent(ctx, "dev-d", 4)
			var sum float64
			for _, r := range readings {
				sum += r.TempC
			}
			_ = sum
		}
	}()
	wg.Wait()
}

// TestRecentIsolatedSnapshot verifies mutating the slice returned by Recent
// never corrupts the stored history.
func TestRecentIsolatedSnapshot(t *testing.T) {
	ctx := context.Background()
	monitor := NewMonitor(8, policyForDispatchTest())
	for i := 0; i < 5; i++ {
		_, _ = monitor.ProcessReading(ctx, Reading{DeviceID: "dev-e", TempC: float64(i)})
	}
	recent, err := monitor.Recent(ctx, "dev-e", 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := range recent {
		recent[i].TempC = 999
	}
	again, err := monitor.Recent(ctx, "dev-e", 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range again {
		if r.TempC == 999 {
			t.Fatal("mutating the recent slice leaked into the stored history")
		}
	}
}
