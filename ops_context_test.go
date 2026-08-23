package main

import (
	"context"
	"errors"
	"testing"
)

// TestOpsPutCanceledDoesNotCommit verifies a canceled put never stores the
// record even though it returns an error.
func TestOpsPutCanceledDoesNotCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := newOpsStore(nil)
	err := store.Put(ctx, OpsRecord{ID: "p1", Subject: "x", Owner: "alice", Priority: OpsPriorityNormal, Labels: map[string]string{"site": "west"}})
	if err == nil {
		t.Fatal("expected a context error from Put")
	}
	if _, err := store.Get(context.Background(), "p1"); !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("canceled put must not commit, got %v", err)
	}
}

// TestOpsUpdateCanceledDoesNotCommit verifies a canceled update leaves the
// stored revision untouched.
func TestOpsUpdateCanceledDoesNotCommit(t *testing.T) {
	store := newOpsStore([]OpsRecord{{ID: "u1", Subject: "s", Owner: "alice", Priority: OpsPriorityNormal, Labels: map[string]string{"site": "west"}}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := store.Update(ctx, OpsRecord{ID: "u1", Subject: "changed", Owner: "alice", Priority: OpsPriorityNormal, Labels: map[string]string{"site": "west"}}, 1)
	if err == nil {
		t.Fatal("expected a context error from Update")
	}
	record, err := store.Get(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if record.Revision != 1 {
		t.Fatalf("canceled update must not bump the revision, got %d", record.Revision)
	}
}

// TestOpsDeleteCanceledDoesNotCommit verifies a canceled delete keeps the
// record in the store.
func TestOpsDeleteCanceledDoesNotCommit(t *testing.T) {
	store := newOpsStore([]OpsRecord{{ID: "d1", Subject: "s", Owner: "alice", Priority: OpsPriorityNormal, Labels: map[string]string{"site": "west"}}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Delete(ctx, "d1"); err == nil {
		t.Fatal("expected a context error from Delete")
	}
	if _, err := store.Get(context.Background(), "d1"); err != nil {
		t.Fatalf("canceled delete must not remove the record, got %v", err)
	}
}

// TestSweepHaltsWhenCanceled verifies a canceled sweep stops between items
// instead of continuing to transition records.
func TestSweepHaltsWhenCanceled(t *testing.T) {
	svc := newOpsService([]OpsRecord{{ID: "r1", Subject: "s", Owner: "alice", Priority: OpsPriorityNormal, Labels: map[string]string{"site": "west"}}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done, err := Reconcile(ctx, svc, []string{"r1"}, OpsStatusActive)
	if err == nil {
		t.Fatalf("expected a context error, got done=%d", done)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if done != 0 {
		t.Fatalf("canceled sweep must not process items, got %d", done)
	}
}
