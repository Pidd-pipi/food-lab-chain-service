package intake

import (
	"context"
	"errors"
	"testing"
	"time"
)

func seedValidBatch(t *testing.T, repo *Repo, id string) {
	t.Helper()
	batch := Batch{
		ID:        id,
		SourceSite: "site-x",
		Collector: "alice",
		Items:     []Item{{ID: "i1", SampleCode: "S-1", FoodType: "rice", Quantity: 1}},
	}
	if err := repo.Save(context.Background(), &batch); err != nil {
		t.Fatal(err)
	}
}

// TestProcessAllReleasesLimiter verifies the limiter slot is returned after
// each batch so a sweep with more batches than slots never stalls.
func TestProcessAllReleasesLimiter(t *testing.T) {
	ctx := context.Background()
	repo := NewRepo()
	for _, id := range []string{"b1", "b2", "b3", "b4"} {
		seedValidBatch(t, repo, id)
	}
	processor := NewProcessor(repo, DefaultPolicy(), 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = processor.ProcessAll(ctx, []string{"b1", "b2", "b3", "b4"})
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ProcessAll hung: limiter slots were never released mid-run")
	}
}

// TestReconcileReleasesOnError verifies a failed batch still returns its
// limiter slot so later sweeps can proceed.
func TestReconcileReleasesOnError(t *testing.T) {
	ctx := context.Background()
	repo := NewRepo()
	seedValidBatch(t, repo, "good-1")
	bad := Batch{ID: "bad-1", SourceSite: "site-x", Items: []Item{{ID: "i1", SampleCode: "S-1", FoodType: "rice", Quantity: 1}}}
	if err := repo.Save(ctx, &bad); err != nil {
		t.Fatal(err)
	}
	processor := NewProcessor(repo, DefaultPolicy(), 1)
	if _, err := processor.Reconcile(ctx, []string{"good-1", "bad-1"}); err == nil {
		t.Fatal("expected reconcile error on the invalid batch")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = processor.Reconcile(ctx, []string{"good-1"})
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("limiter slot leaked after reconcile error")
	}
}

// TestValidateBatchPreservesError verifies ValidateBatch keeps the original
// policy error chain instead of replacing it with a generic message.
func TestValidateBatchPreservesError(t *testing.T) {
	batch := Batch{
		ID:    "b9",
		Items: []Item{{ID: "i1", SampleCode: "S-1", FoodType: "rice", Quantity: 1}},
	}
	ok, err := ValidateBatch(batch)
	if ok {
		t.Fatal("expected the batch to be invalid")
	}
	if !errors.Is(err, ErrCollectorRequired) {
		t.Fatalf("expected ErrCollectorRequired, got %v", err)
	}
}
