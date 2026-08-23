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
func (s *Store) List() []domain.Specimen {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Specimen, 0, len(s.specimens))
	for _, item := range s.specimens {
		result = append(result, *item)
	}
	return result
}
func (s *Store) Handoff(id, recipient string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	specimen, ok := s.specimens[id]
	if !ok {
		return domain.ErrSpecimenNotFound
	}
	return domain.Handoff(specimen, recipient)
}
