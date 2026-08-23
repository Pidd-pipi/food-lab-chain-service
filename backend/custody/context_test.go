package custody

import (
	"context"
	"errors"
	"testing"
)

func traceEventsForTest() []Event {
	return []Event{
		{ID: "e1", SpecimenID: "sp-1", FoodType: "leafy greens", Actor: "field", Action: "collected", At: "2026-08-20T09:00:00Z"},
		{ID: "e2", SpecimenID: "sp-1", FoodType: "leafy greens", Actor: "courier", Action: "transferred", At: "2026-08-20T11:00:00Z"},
		{ID: "e3", SpecimenID: "sp-1", FoodType: "leafy greens", Actor: "lab", Action: "received", At: "2026-08-20T13:00:00Z"},
		{ID: "e4", SpecimenID: "sp-2", FoodType: "cheese", Actor: "field", Action: "collected", At: "2026-08-20T08:00:00Z"},
		{ID: "e5", SpecimenID: "sp-1", FoodType: "leafy greens", Actor: "lab", Action: "sealed", At: "2026-08-20T15:00:00Z"},
	}
}

// TestBuildHonorsContextCancel verifies a canceled build returns the context
// error instead of a half-assembled chain.
func TestBuildHonorsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	builder := NewBuilder(traceEventsForTest())
	chain, err := builder.Build(ctx, "sp-1")
	if err == nil {
		t.Fatal("expected context error, got a chain")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if chain != nil {
		t.Fatalf("canceled build must not return a chain")
	}
}

// TestBuildAllStopsOnCancel verifies a canceled bulk build stops immediately.
func TestBuildAllStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	builder := NewBuilder(traceEventsForTest())
	chains, err := builder.BuildAll(ctx, []string{"sp-1", "sp-2"})
	if err == nil {
		t.Fatal("expected context error from BuildAll")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(chains) != 0 {
		t.Fatalf("canceled BuildAll must not return partial chains, got %d", len(chains))
	}
}

// TestFilterHonorsContext verifies filtering stops when the context is
// canceled instead of returning a filtered result.
func TestFilterHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	builder := NewBuilder(traceEventsForTest())
	chain, err := builder.Build(context.Background(), "sp-1")
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := Filter(ctx, []Chain{*chain}, Query{FoodType: "leafy greens"})
	if err == nil {
		t.Fatal("expected context error from Filter")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("canceled Filter must not return items, got %d", len(filtered))
	}
}

// TestPaginateHonorsContext verifies pagination reports cancellation.
func TestPaginateHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	builder := NewBuilder(traceEventsForTest())
	chain, err := builder.Build(context.Background(), "sp-1")
	if err != nil {
		t.Fatal(err)
	}
	page, err := Paginate(ctx, []Chain{*chain}, Query{Page: 1, PageSize: 10})
	if err == nil {
		t.Fatal("expected context error from Paginate")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("canceled Paginate must not return items, got %d", page.Total)
	}
}

// TestGroupByFoodTypeHonorsContext verifies grouping stops when the context is
// canceled instead of returning a partial grouping.
func TestGroupByFoodTypeHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	builder := NewBuilder(traceEventsForTest())
	chain, err := builder.Build(context.Background(), "sp-1")
	if err != nil {
		t.Fatal(err)
	}
	grouped, err := GroupByFoodType(ctx, []Chain{*chain})
	if err == nil {
		t.Fatal("expected context error from GroupByFoodType")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(grouped) != 0 {
		t.Fatalf("canceled grouping must not return items, got %d", len(grouped))
	}
}
