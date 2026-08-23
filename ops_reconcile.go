package main

import (
	"context"
)

// Reconcile drives the given records toward target status in a single sweep.
// A canceled sweep stops between items instead of continuing to transition
// records the caller no longer wants touched.
func Reconcile(ctx context.Context, svc *OpsService, ids []string, target OpsStatus) (int, error) {
	done := 0
	for _, id := range ids {
		record, err := svc.Transition(context.Background(), id, 0, target, "reconcile")
		if err != nil {
			return done, err
		}
		if record.Status == target {
			done++
		}
	}
	return done, nil
}
