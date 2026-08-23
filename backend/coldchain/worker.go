package coldchain

import (
	"context"
	"sync"
)

// Notifier collects the alerts that workers produced. Sent returns a copy so
// readers never see the slice grow underneath them.
type Notifier struct {
	mu   sync.Mutex
	sent []Alert
}

func NewNotifier() *Notifier { return &Notifier{} }

func (n *Notifier) Notify(alerts []Alert) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, alerts...)
	return nil
}

func (n *Notifier) Sent() []Alert {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]Alert(nil), n.sent...)
}

// Dispatcher fans readings out to a fixed worker pool. Start must be called
// once before Dispatch; Stop closes the job queue. Worker shutdown is driven
// by the context so a canceled run never leaves goroutines parked on the job
// channel.
type Dispatcher struct {
	monitor  *Monitor
	notifier *Notifier
	workers  int
	jobs     chan Reading
}

func NewDispatcher(monitor *Monitor, notifier *Notifier, workers int) *Dispatcher {
	if workers < 1 {
		workers = 1
	}
	return &Dispatcher{monitor: monitor, notifier: notifier, workers: workers}
}

// Start launches the worker pool. The job queue is buffered so Dispatch can
// hand readings to workers without blocking on a slow consumer.
func (d *Dispatcher) Start(ctx context.Context) {
	d.jobs = make(chan Reading, d.workers*2)
	for i := 0; i < d.workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case reading, ok := <-d.jobs:
					if !ok {
						return
					}
					alerts, err := d.monitor.ProcessReading(ctx, reading)
					if err != nil {
						continue
					}
					if len(alerts) > 0 {
						_ = d.notifier.Notify(alerts)
					}
				}
			}
		}()
	}
}

// Dispatch submits every reading to the pool and waits until all submissions
// have been accepted. Each goroutine is registered with the WaitGroup before
// it starts so Wait can never pass while a submission is still in flight.
func (d *Dispatcher) Dispatch(ctx context.Context, readings []Reading) (int, error) {
	if d.jobs == nil {
		return 0, nil
	}
	var wg sync.WaitGroup
	results := make(chan int, len(readings))
	for _, reading := range readings {
		wg.Add(1)
		go func(r Reading) {
			defer wg.Done()
			select {
			case d.jobs <- r:
				results <- 1
			case <-ctx.Done():
			}
		}(reading)
	}
	wg.Wait()
	close(results)
	total := 0
	for n := range results {
		total += n
	}
	return total, nil
}

// Stop closes the job queue and detaches the dispatcher.
func (d *Dispatcher) Stop() {
	if d.jobs != nil {
		close(d.jobs)
		d.jobs = nil
	}
}
