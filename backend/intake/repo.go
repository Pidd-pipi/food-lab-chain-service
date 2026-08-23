package intake

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrBatchNotFound    = errors.New("batch not found")
	ErrRevisionConflict = errors.New("revision conflict")
)

// Repo stores batches in insertion order. Every read path hands back clones so
// a caller cannot corrupt the in-memory state by mutating a returned batch.
type Repo struct {
	mu      sync.RWMutex
	batches map[string]*Batch
	order   []string
}

func NewRepo() *Repo {
	return &Repo{batches: map[string]*Batch{}}
}

func (r *Repo) Save(ctx context.Context, batch *Batch) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := batch.Clone()
	if _, ok := r.batches[clone.ID]; !ok {
		r.order = append(r.order, clone.ID)
	}
	r.batches[clone.ID] = clone
	return nil
}

func (r *Repo) Get(ctx context.Context, id string) (*Batch, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	batch, ok := r.batches[id]
	if !ok {
		return nil, ErrBatchNotFound
	}
	return batch.Clone(), nil
}

// List returns all batches in insertion order. The returned slice and each
// batch are independent copies of the internal state.
func (r *Repo) List(ctx context.Context) ([]*Batch, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Batch, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.batches[id].Clone())
	}
	return out, nil
}

func (r *Repo) Update(ctx context.Context, batch *Batch, expected int) (*Batch, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.batches[batch.ID]
	if !ok {
		return nil, ErrBatchNotFound
	}
	if expected > 0 && current.Revision != expected {
		return nil, ErrRevisionConflict
	}
	clone := batch.Clone()
	clone.Revision = current.Revision + 1
	r.batches[clone.ID] = clone
	return clone.Clone(), nil
}
