package custody

import (
	"context"
	"errors"
	"sort"
)

var ErrChainNotFound = errors.New("chain not found")

// Event is a raw custody event fed into the builder.
type Event struct {
	ID         string
	SpecimenID string
	FoodType   string
	Actor      string
	Action     string
	At         string
}

// Builder assembles custody chains from a chronological event stream. Build
// honors the context: a canceled build stops immediately and never returns a
// half-assembled chain as if it were complete.
type Builder struct {
	events []Event
}

func NewBuilder(events []Event) *Builder {
	sorted := append([]Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At < sorted[j].At })
	return &Builder{events: sorted}
}

func (b *Builder) Build(ctx context.Context, specimenID string) (*Chain, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	chain := &Chain{SpecimenID: specimenID}
	step := 1
	for _, event := range b.events {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if event.SpecimenID != specimenID {
			continue
		}
		chain.Nodes = append(chain.Nodes, Node{
			Step:       step,
			EventID:    event.ID,
			SpecimenID: event.SpecimenID,
			FoodType:   event.FoodType,
			Actor:      event.Actor,
			Action:     event.Action,
			At:         event.At,
		})
		step++
	}
	if len(chain.Nodes) == 0 {
		return nil, ErrChainNotFound
	}
	if last := chain.Nodes[len(chain.Nodes)-1]; last.Action == "delivered" || last.Action == "sealed" {
		chain.Complete = true
	}
	return chain, nil
}

// BuildAll assembles the requested chains sequentially and stops as soon as
// the context is canceled. On cancellation it returns no chains rather than a
// partial slice, so a half-built batch can never be mistaken for a finished
// one downstream.
func (b *Builder) BuildAll(ctx context.Context, ids []string) ([]Chain, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	chains := make([]Chain, 0, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		chain, err := b.Build(ctx, id)
		if err != nil {
			if errors.Is(err, ErrChainNotFound) {
				continue
			}
			return nil, err
		}
		chains = append(chains, *chain)
	}
	return chains, nil
}
