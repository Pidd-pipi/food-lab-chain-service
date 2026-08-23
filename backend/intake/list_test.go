package intake

import (
	"context"
	"testing"
)

func seedBatches(t *testing.T, repo *Repo, svc *Service) {
	t.Helper()
	for _, input := range []Batch{
		{ID: "b1", SourceSite: "site-a", Collector: "alice", Items: []Item{{ID: "i1", SampleCode: "S-1", FoodType: "leafy greens", Quantity: 2}}},
		{ID: "b2", SourceSite: "site-b", Collector: "bob", Items: []Item{{ID: "i2", SampleCode: "S-2", FoodType: "cheese", Quantity: 3}}},
	} {
		if _, err := svc.Create(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
}

// TestListReturnsIndependentSlice verifies that mutating a batch returned by
// List never changes what the store serves on the next read.
func TestListReturnsIndependentSlice(t *testing.T) {
	ctx := context.Background()
	repo := NewRepo()
	svc := NewService(repo)
	seedBatches(t, repo, svc)

	first, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(first))
	}
	first[0].Status = StatusApproved
	first[0].Items = append(first[0].Items, Item{ID: "x", SampleCode: "S-X", FoodType: "rice", Quantity: 1})

	second, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second[0].Status != StatusDraft {
		t.Fatalf("mutating a listed batch leaked into the store: status=%s", second[0].Status)
	}
	if len(second[0].Items) != 1 {
		t.Fatalf("mutating a listed batch leaked items into the store: items=%d", len(second[0].Items))
	}
}

// TestFilterDoesNotCorruptOtherBatches verifies that filtering one status in
// place cannot reorder or drop batches that are visible to later list calls.
func TestFilterDoesNotCorruptOtherBatches(t *testing.T) {
	ctx := context.Background()
	repo := NewRepo()
	svc := NewService(repo)
	// An approved batch is inserted first so an in-place compaction has to
	// overwrite it while building the draft result. Seeding goes through
	// repo.Save so statuses survive (Create always starts batches as draft).
	for _, b := range []Batch{
		{ID: "a1", SourceSite: "s1", Collector: "alice", Status: StatusApproved},
		{ID: "a2", SourceSite: "s2", Collector: "bob", Status: StatusDraft},
		{ID: "a3", SourceSite: "s3", Collector: "carol", Status: StatusDraft},
	} {
		if err := repo.Save(ctx, &b); err != nil {
			t.Fatal(err)
		}
	}

	drafts, err := svc.FilterByStatus(ctx, StatusDraft)
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 2 || drafts[0].ID != "a2" || drafts[1].ID != "a3" {
		t.Fatalf("unexpected draft filter result: %+v", drafts)
	}

	all, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("filtering corrupted the batch listing: got %d batches, want 3", len(all))
	}
	ids := map[string]bool{}
	for _, b := range all {
		ids[b.ID] = true
	}
	for _, want := range []string{"a1", "a2", "a3"} {
		if !ids[want] {
			t.Fatalf("batch %s missing after filtering: %v", want, all)
		}
	}
}

// TestAppendItemNoAlias verifies that appending to a batch fetched from the
// repo does not write through to the stored batch until Update is called.
func TestAppendItemNoAlias(t *testing.T) {
	ctx := context.Background()
	repo := NewRepo()
	svc := NewService(repo)
	seedBatches(t, repo, svc)

	got, err := repo.Get(ctx, "b1")
	if err != nil {
		t.Fatal(err)
	}
	got.Items = append(got.Items, Item{ID: "sneaky", SampleCode: "S-9", FoodType: "nuts", Quantity: 5})

	fresh, err := repo.Get(ctx, "b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh.Items) != 1 {
		t.Fatalf("appending to a fetched batch leaked into the store before Update: items=%d", len(fresh.Items))
	}
}

// TestBatchCloneDeepCopiesItems verifies Clone isolates the Items slice.
func TestBatchCloneDeepCopiesItems(t *testing.T) {
	batch := &Batch{
		ID:      "b9",
		Status:  StatusDraft,
		Items:   []Item{{ID: "i1", SampleCode: "S-1", FoodType: "meat", Quantity: 1}},
	}
	clone := batch.Clone()
	clone.Items[0].SampleCode = "CHANGED"
	clone.Items = append(clone.Items, Item{ID: "i2", SampleCode: "S-2", FoodType: "fish", Quantity: 2})
	if batch.Items[0].SampleCode != "S-1" {
		t.Fatalf("clone mutation leaked into original: %s", batch.Items[0].SampleCode)
	}
	if len(batch.Items) != 1 {
		t.Fatalf("clone append leaked into original: items=%d", len(batch.Items))
	}
}

// TestSaveIsolation verifies a batch saved through the repo is stored as an
// independent copy; mutating the caller's struct must not leak into the store.
func TestSaveIsolation(t *testing.T) {
	ctx := context.Background()
	repo := NewRepo()
	batch := Batch{ID: "iso-1", SourceSite: "s1", Collector: "alice", Items: []Item{{ID: "i1", SampleCode: "S-1", FoodType: "rice", Quantity: 1}}}
	if err := repo.Save(ctx, &batch); err != nil {
		t.Fatal(err)
	}
	batch.Items = append(batch.Items, Item{ID: "i2", SampleCode: "S-2", FoodType: "fish", Quantity: 2})
	batch.Collector = "carol"
	got, err := repo.Get(ctx, "iso-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Collector != "alice" {
		t.Fatalf("mutation after Save leaked into the store: collector=%s", got.Collector)
	}
	if len(got.Items) != 1 {
		t.Fatalf("mutation after Save leaked items into the store: items=%d", len(got.Items))
	}
}

// TestDeleteRemovesFromListing verifies a deleted batch disappears from List
// as well as from direct lookups.
func TestDeleteRemovesFromListing(t *testing.T) {
	ctx := context.Background()
	repo := NewRepo()
	for _, id := range []string{"d1", "d2", "d3"} {
		batch := Batch{ID: id, SourceSite: "s1", Collector: "alice"}
		if err := repo.Save(ctx, &batch); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Delete(ctx, "d2"); err != nil {
		t.Fatal(err)
	}
	all, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range all {
		if b.ID == "d2" {
			t.Fatal("deleted batch still appears in List")
		}
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 batches after delete, got %d", len(all))
	}
}
