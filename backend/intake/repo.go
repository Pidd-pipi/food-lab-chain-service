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
	clone := batch.Clone()
	if _, ok := r.byID[batch.ID]; !ok {
		r.ordered = append(r.ordered, clone)
	} else {
		for i, item := range r.ordered {
			if item.ID == clone.ID {
				r.ordered[i] = clone
				break
			}
		}
	}
	r.byID[batch.ID] = clone
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
	return batch.Clone(), nil
}

// List returns all batches in insertion order. The returned slice and each
// batch (including its specimen lines) are independent copies of the internal
// state, so callers may filter, reorder, or mutate them without disturbing the
// in-memory store.
func (r *Repo) List(ctx context.Context) ([]*Batch, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Batch, 0, len(r.ordered))
	for _, batch := range r.ordered {
		out = append(out, batch.Clone())
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
	for i, item := range r.ordered {
		if item.ID == id {
			r.ordered = append(r.ordered[:i], r.ordered[i+1:]...)
			break
		}
	}
	return nil
}
