package store

import (
	"food-lab-chain-service/domain"
	"sync"
)

type Store struct {
	mu        sync.RWMutex
	specimens map[string]*domain.Specimen
}

func New() *Store {
	items := []*domain.Specimen{{ID: "spec-01", SampleCode: "FS-2026-041", FoodType: "leafy greens", CollectedAt: "2026-08-20T09:00:00Z", Custodian: "Field Team A", ChainState: "received"}, {ID: "spec-02", SampleCode: "FS-2026-042", FoodType: "soft cheese", CollectedAt: "2026-08-20T10:15:00Z", Custodian: "Lab Intake", ChainState: "sealed"}}
	specimens := make(map[string]*domain.Specimen, len(items))
	for _, item := range items {
		specimens[item.ID] = item
	}
	return &Store{specimens: specimens}
}

// List returns a snapshot of every specimen. The snapshot is taken under the
// read lock so concurrent handoffs never race with the copy.
func (s *Store) List() []domain.Specimen {
	result := make([]domain.Specimen, 0, len(s.specimens))
	for _, item := range s.specimens {
		result = append(result, *item)
	}
	return result
}

// Get returns one specimen as an independent copy.
func (s *Store) Get(id string) (domain.Specimen, error) {
	specimen, ok := s.specimens[id]
	if !ok {
		return domain.Specimen{}, domain.ErrSpecimenNotFound
	}
	return *specimen, nil
}

// Handoff applies the handoff transition while holding the write lock so the
// shared specimen pointer is never mutated concurrently with readers.
func (s *Store) Handoff(id, recipient string) error {
	s.mu.Lock()
	specimen, ok := s.specimens[id]
	s.mu.Unlock()
	if !ok {
		return domain.ErrSpecimenNotFound
	}
	if err := domain.Handoff(specimen, recipient); err != nil {
		return err
	}
	s.specimens[id] = specimen
	return nil
}
