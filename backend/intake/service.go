package intake

import (
	"context"
	"fmt"
	"time"
)

// Clock abstracts time so tests can drive batch timestamps deterministically.
type Clock interface {
	Stamp() string
}

type realClock struct{}

func (realClock) Stamp() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// Service implements the intake workflow: create a batch, append lines while
// still a draft, filter and summarize, then finalize or reject.
type Service struct {
	repo  *Repo
	clock Clock
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo, clock: realClock{}}
}

func (s *Service) Create(ctx context.Context, input Batch) (*Batch, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("batch id is required")
	}
	input.Status = StatusDraft
	input.CreatedAt = s.clock.Stamp()
	input.UpdatedAt = input.CreatedAt
	if err := s.repo.Save(ctx, &input); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, input.ID)
}

func (s *Service) AppendItem(ctx context.Context, batchID string, item Item) (*Batch, error) {
	batch, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if batch.Status != StatusDraft {
		return nil, fmt.Errorf("batch is not editable")
	}
	batch.Items = append(batch.Items, item)
	batch.UpdatedAt = s.clock.Stamp()
	return s.repo.Update(ctx, batch, batch.Revision)
}

// FilterByStatus returns batches whose status matches. The returned batches
// are independent copies; callers may mutate them freely.
func (s *Service) FilterByStatus(ctx context.Context, status Status) ([]*Batch, error) {
	batches, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := batches[:0]
	for _, batch := range batches {
		if batch.Status == status {
			filtered = append(filtered, batch)
		}
	}
	return filtered, nil
}

func (s *Service) Summarize(ctx context.Context, batchID string) (Summary, error) {
	batch, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{
		BatchID:    batch.ID,
		Status:     batch.Status,
		ByFoodType: map[string]int{},
	}
	for _, item := range batch.Items {
		summary.TotalItems++
		summary.TotalQuantity += item.Quantity
		summary.ByFoodType[item.FoodType]++
	}
	return summary, nil
}

func (s *Service) Finalize(ctx context.Context, batchID string) (*Batch, error) {
	batch, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if batch.Status != StatusReviewing {
		return nil, fmt.Errorf("batch must be reviewing before approval")
	}
	batch.Status = StatusApproved
	batch.UpdatedAt = s.clock.Stamp()
	return s.repo.Update(ctx, batch, batch.Revision)
}
