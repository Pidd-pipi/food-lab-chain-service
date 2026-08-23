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

// jobBuffer caps how many readings can sit waiting for a worker. Keeping it
// bounded means a big sweep is throttled by the worker pool instead of
// buffering the whole batch (and its goroutines) in memory.
const jobBuffer = 64

func NewDispatcher(monitor *Monitor, notifier *Notifier, workers int) *Dispatcher {
	if workers < 1 {
		workers = 1
	}
	return &Dispatcher{monitor: monitor, notifier: notifier, workers: workers}
}

// Start launches the worker pool. The job queue is buffered so Dispatch can
// hand readings to workers without blocking on a slow consumer. Each worker
// selects on the run context as well as the job channel, so canceling the
// context the dispatcher was started with drains the pool promptly instead of
// leaving workers parked until Stop closes the channel.
func (d *Dispatcher) Start(ctx context.Context) {
	jobs := make(chan Reading, jobBuffer)
	d.jobs = jobs
	for i := 0; i < d.workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case reading, ok := <-jobs:
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

// Dispatch submits every reading to the pool and returns how many were handed
// off. Submission is single-threaded and bounded by the job buffer, so a sweep
// of any size is processed at worker-pool concurrency rather than spawning a
// goroutine per reading. The select on ctx.Done() lets a canceled dispatch
// stop immediately instead of blocking forever on a full queue or racing a
// separate result channel.
func (d *Dispatcher) Dispatch(ctx context.Context, readings []Reading) (int, error) {
	if d.jobs == nil {
		return 0, nil
	}
	total := 0
	for _, reading := range readings {
		select {
		case d.jobs <- reading:
			total++
		case <-ctx.Done():
			return total, ctx.Err()
		}
	}
	return total, nil
}

// Stop closes the job queue and detaches the dispatcher. It must be called
// only after every in-flight Dispatch has returned; closing the channel while a
// dispatch is still sending would panic.
func (d *Dispatcher) Stop() {
	if d.jobs != nil {
		close(d.jobs)
		d.jobs = nil
	}
}
