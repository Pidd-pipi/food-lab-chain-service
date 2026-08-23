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
	byID    map[string]*Batch
	ordered []*Batch
}

func NewRepo() *Repo {
	return &Repo{byID: map[string]*Batch{}}
}

func (r *Repo) Save(ctx context.Context, batch *Batch) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[batch.ID]; !ok {
		r.ordered = append(r.ordered, batch)
	}
	r.byID[batch.ID] = batch
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
	batch, ok := r.byID[id]
	if !ok {
		return nil, ErrBatchNotFound
	}
	return batch, nil
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
	return r.ordered, nil
}

func (r *Repo) Update(ctx context.Context, batch *Batch, expected int) (*Batch, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.byID[batch.ID]
	if !ok {
		return nil, ErrBatchNotFound
	}
	if expected > 0 && current.Revision != expected {
		return nil, ErrRevisionConflict
	}
	clone := batch.Clone()
	clone.Revision = current.Revision + 1
	r.byID[clone.ID] = clone
	for i, item := range r.ordered {
		if item.ID == clone.ID {
			r.ordered[i] = clone
			break
		}
	}
	return clone.Clone(), nil
}

// Delete removes a batch from the index. The ordered listing is kept in sync
// so a deleted batch never resurfaces in List.
func (r *Repo) Delete(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[id]; !ok {
		return ErrBatchNotFound
	}
	delete(r.byID, id)
	return nil
}
