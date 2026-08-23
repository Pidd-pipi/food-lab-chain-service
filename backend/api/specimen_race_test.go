package api

import (
	"encoding/json"
	"food-lab-chain-service/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestHandoffResponseConfirmsState verifies a successful handoff response
// reports the new chain state so callers can confirm the transition.
func TestHandoffResponseConfirmsState(t *testing.T) {
	server := httptest.NewServer(routeForTest())
	defer server.Close()
	resp, err := http.Post(server.URL+"/api/specimens/spec-01/handoff", "application/json", strings.NewReader(`{"to":"Microbiology Bench"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("handoff status: %d", resp.StatusCode)
	}
	var body struct {
		Status     string `json:"status"`
		ChainState string `json:"chainState"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.ChainState != "in-transit" {
		t.Fatalf("expected chainState in-transit in response, got %q", body.ChainState)
	}
}

// TestConcurrentListAndHandoffHttp drives the list endpoint and handoff
// endpoint concurrently: every request must succeed and no shared state may be
// torn.
func TestConcurrentListAndHandoffHttp(t *testing.T) {
	server := httptest.NewServer(routeForTest())
	defer server.Close()
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			resp, err := http.Get(server.URL + "/api/specimens")
			if err != nil {
				t.Error(err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("list status: %d", resp.StatusCode)
			}
			resp.Body.Close()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			resp, err := http.Post(server.URL+"/api/specimens/spec-01/handoff", "application/json", strings.NewReader(`{"to":"Microbiology Bench"}`))
			if err != nil {
				t.Error(err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("handoff status: %d", resp.StatusCode)
			}
			resp.Body.Close()
		}
	}()
	close(start)
	wg.Wait()
}

// TestConcurrentGetAndHandoff drives the store read and write paths directly
// to catch races on the shared specimen pointer.
func TestConcurrentGetAndHandoff(t *testing.T) {
	s := store.New()
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_, _ = s.Get("spec-01")
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_, _ = s.Handoff("spec-01", "Microbiology Bench")
		}
	}()
	close(start)
	wg.Wait()
}
