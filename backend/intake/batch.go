package intake

import (
	"context"
	"fmt"
)

// BatchResult reports the outcome of processing a single batch.
type BatchResult struct {
	BatchID   string
	Processed int
	Err       error
}

// Processor runs batch jobs under a concurrency limiter so a big sweep cannot
// open unlimited parallel work. Every acquire must be paired with a release on
// all paths, including error returns.
type Processor struct {
	repo    *Repo
	policy  Policy
	limiter chan struct{}
}

func NewProcessor(repo *Repo, policy Policy, maxParallel int) *Processor {
	if maxParallel < 1 {
		maxParallel = 1
	}
	return &Processor{repo: repo, policy: policy, limiter: make(chan struct{}, maxParallel)}
}

func (p *Processor) acquire(ctx context.Context) error {
	select {
	case p.limiter <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Processor) release() {
	<-p.limiter
}

func (p *Processor) ProcessBatch(ctx context.Context, batchID string) (BatchResult, error) {
	var result BatchResult
	result.BatchID = batchID
	batch, err := p.repo.Get(ctx, batchID)
	if err != nil {
		result.Err = err
		return result, err
	}
	if err := p.policy.CheckBatch(*batch); err != nil {
		result.Err = err
		return result, err
	}
	if errs := EvaluateAll(batch.Items, p.policy); len(errs) > 0 {
		result.Err = fmt.Errorf("batch %s has %d invalid items", batchID, len(errs))
		return result, result.Err
	}
	result.Processed = len(batch.Items)
	return result, nil
}

// ProcessAll sweeps the given batches, releasing the limiter slot after each
// batch so the pool never fills up mid-run.
func (p *Processor) ProcessAll(ctx context.Context, ids []string) ([]BatchResult, error) {
	results := make([]BatchResult, 0, len(ids))
	for _, id := range ids {
		if err := p.acquire(ctx); err != nil {
			return results, err
		}
		defer p.release()
		result, err := p.ProcessBatch(ctx, id)
		results = append(results, result)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// Reconcile revalidates a sweep and counts the batches that still pass. The
// limiter slot is released on both success and error paths.
func (p *Processor) Reconcile(ctx context.Context, ids []string) (int, error) {
	processed := 0
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		if err := p.acquire(ctx); err != nil {
			return processed, err
		}
		if _, err := p.ProcessBatch(ctx, id); err != nil {
			return processed, err
		}
		p.release()
		processed++
	}
	return processed, nil
}
