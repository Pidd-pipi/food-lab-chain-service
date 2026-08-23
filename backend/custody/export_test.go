package custody

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type limitedWriter struct {
	limit   int
	written int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.written+len(p) > w.limit {
		return 0, errors.New("output exceeds limit")
	}
	w.written += len(p)
	return len(p), nil
}

// TestExportAllReportsWriteError verifies an I/O failure during streaming is
// surfaced to the caller and never swallowed by the final flush.
func TestExportAllReportsWriteError(t *testing.T) {
	ctx := context.Background()
	events := make([]Event, 0, 500)
	for i := 0; i < 100; i++ {
		for step := 0; step < 5; step++ {
			events = append(events, Event{
				ID:         fmt.Sprintf("ev-%d-%d", i, step),
				SpecimenID: fmt.Sprintf("sp-%d", i),
				FoodType:   "leafy greens",
				Actor:      "lab",
				Action:     "handled",
				At:         fmt.Sprintf("2026-08-20T%02d:00:00Z", step),
			})
		}
	}
	builder := NewBuilder(events)
	chains := make([]Chain, 0, 100)
	for i := 0; i < 100; i++ {
		chain, err := builder.Build(ctx, fmt.Sprintf("sp-%d", i))
		if err != nil {
			t.Fatal(err)
		}
		chains = append(chains, *chain)
	}
	writer := &limitedWriter{limit: 4096}
	written, err := ExportAll(ctx, writer, chains)
	if err == nil {
		t.Fatalf("expected a write error, got nil (written=%d)", written)
	}
}
