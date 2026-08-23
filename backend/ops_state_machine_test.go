package main

import (
	"context"
	"testing"
)

// TestTransitionReviewingToClosed verifies a reviewing record can be moved to
// closed; the transition table must not strand it in the intermediate state.
func TestTransitionReviewingToClosed(t *testing.T) {
	sm := newOpsStateMachine()
	if err := sm.Move(OpsStatusReviewing, OpsStatusClosed, "review passed"); err != nil {
		t.Fatalf("reviewing -> closed should be allowed, got %v", err)
	}
	last, ok := sm.Last()
	if !ok || last.To != OpsStatusClosed {
		t.Fatalf("expected last transition to closed, got %+v", last)
	}
}

// TestTransitionActiveToReviewing verifies a record can enter the reviewing
// state from active.
func TestTransitionActiveToReviewing(t *testing.T) {
	sm := newOpsStateMachine()
	if err := sm.Move(OpsStatusActive, OpsStatusReviewing, "needs review"); err != nil {
		t.Fatalf("active -> reviewing should be allowed, got %v", err)
	}
}

// TestReviewingStatusValid verifies reviewing is a recognized status.
func TestReviewingStatusValid(t *testing.T) {
	if !opsStatusValid(OpsStatusReviewing) {
		t.Fatal("reviewing must be a valid status")
	}
	if opsStatusTerminal(OpsStatusReviewing) {
		t.Fatal("reviewing must not be a terminal status")
	}
}

// TestActiveQueryIncludesReviewing verifies in-progress filters and queries
// still surface records parked in reviewing.
func TestActiveQueryIncludesReviewing(t *testing.T) {
	if !opsInProgress(OpsStatusReviewing) {
		t.Fatal("reviewing must count as in progress")
	}
	records := []OpsRecord{
		{ID: "r1", Status: OpsStatusQueued},
		{ID: "r2", Status: OpsStatusReviewing},
		{ID: "r3", Status: OpsStatusClosed},
	}
	active := opsFilterInProgress(records)
	if len(active) != 2 {
		t.Fatalf("expected 2 in-progress records, got %d", len(active))
	}
	for _, record := range active {
		if record.ID == "r3" {
			t.Fatal("closed record must not appear in in-progress results")
		}
	}
}

// TestReviewingNotTerminal verifies a reviewing record is not considered
// terminal by the model.
func TestReviewingNotTerminal(t *testing.T) {
	record := OpsRecord{ID: "r9", Status: OpsStatusReviewing}
	if record.Terminal() {
		t.Fatal("reviewing record must not be terminal")
	}
}

// TestServiceTransitionReviewingFlow drives the whole flow through the
// service: queued -> active -> reviewing -> closed.
func TestServiceTransitionReviewingFlow(t *testing.T) {
	svc := newOpsService(nil)
	created, err := svc.Create(context.Background(), OpsRecord{ID: "flow-1", Subject: "audit", Owner: "alice", Priority: OpsPriorityHigh, Labels: map[string]string{"site": "west"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []OpsStatus{OpsStatusActive, OpsStatusReviewing, OpsStatusClosed} {
		created, err = svc.Transition(context.Background(), created.ID, 0, target, "operator")
		if err != nil {
			t.Fatalf("transition to %s failed: %v", target, err)
		}
	}
	if created.Status != OpsStatusClosed {
		t.Fatalf("expected closed at the end, got %s", created.Status)
	}
}

// TestOpsMatchInProgressPseudoStatus verifies a query for the in-progress
// pseudo-status matches reviewing records.
func TestOpsMatchInProgressPseudoStatus(t *testing.T) {
	record := OpsRecord{ID: "q1", Status: OpsStatusReviewing}
	if !opsMatchInProgress(record, OpsQuery{Status: "in_progress"}) {
		t.Fatal("reviewing record must match an in-progress query")
	}
	closed := OpsRecord{ID: "q2", Status: OpsStatusClosed}
	if opsMatchInProgress(closed, OpsQuery{Status: "in_progress"}) {
		t.Fatal("closed record must not match an in-progress query")
	}
}
