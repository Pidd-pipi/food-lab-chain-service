package main

import (
	"context"
	"food-lab-chain-service/coldchain"
	"testing"
)

// TestEvaluateWithoutPolicyDoesNotPanic verifies that a device without any
// configured policy is skipped instead of crashing on a nil policy dereference.
func TestEvaluateWithoutPolicyDoesNotPanic(t *testing.T) {
	evaluator := coldchain.NewEvaluator(nil) // fleet default is intentionally absent
	alerts, err := evaluator.Evaluate(coldchain.Reading{DeviceID: "dev-nil", TempC: 30.0})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts when no policy exists, got %d", len(alerts))
	}
}

// TestSetLabelInitializesMap verifies label writes never hit a nil map.
func TestSetLabelInitializesMap(t *testing.T) {
	registry := coldchain.NewRegistry()
	if err := registry.Register(coldchain.Device{ID: "dev-1", Name: "freezer-1", Status: "online"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.SetLabel("dev-1", "quarantine"); err != nil {
		t.Fatal(err)
	}
	if err := registry.SetLabel("dev-1", "audited"); err != nil {
		t.Fatal(err)
	}
}

// TestSummarizeEmptyReturnsAllocatedMap verifies the summarized map is always
// writable, even when there are no reports.
func TestSummarizeEmptyReturnsAllocatedMap(t *testing.T) {
	ctx := context.Background()
	byDevice, err := coldchain.Summarize(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	byDevice["dev-new"] = 3 // must not panic on a nil map
	if byDevice["dev-new"] != 3 {
		t.Fatalf("expected written value, got %d", byDevice["dev-new"])
	}
}

// TestSetRuleInitializesMap verifies the first device rule can be registered
// without a nil-map panic.
func TestSetRuleInitializesMap(t *testing.T) {
	evaluator := coldchain.NewEvaluator(nil)
	evaluator.SetRule("dev-rule", coldchain.Policy{WarningHigh: 8, WarningLow: 2, CriticalHigh: 10, CriticalLow: 0})
	alerts, err := evaluator.Evaluate(coldchain.Reading{DeviceID: "dev-rule", TempC: 11.0})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Level != "critical" {
		t.Fatalf("expected one critical alert, got %+v", alerts)
	}
}

// TestRegistryGetReturnsCopy verifies Get hands back an independent copy.
func TestRegistryGetReturnsCopy(t *testing.T) {
	registry := coldchain.NewRegistry()
	if err := registry.Register(coldchain.Device{ID: "dev-copy", Name: "freezer-2", Status: "online"}); err != nil {
		t.Fatal(err)
	}
	device, err := registry.Get("dev-copy")
	if err != nil {
		t.Fatal(err)
	}
	device.Status = "offline"
	fresh, err := registry.Get("dev-copy")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Status != "online" {
		t.Fatalf("mutating a fetched device leaked into the registry: status=%s", fresh.Status)
	}
}
