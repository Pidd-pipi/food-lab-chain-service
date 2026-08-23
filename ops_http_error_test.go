package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func opsErrorTestRouter(svc *OpsService) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/ops/", http.StripPrefix("/api/ops", opsRouter(svc)))
	return mux
}

func createOpsRecord(t *testing.T, router http.Handler, id string) {
	t.Helper()
	body := `{"id":"` + id + `","subject":"daily check","owner":"alice","priority":"high","labels":{"site":"west"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s should be 201, got %d (%s)", id, rec.Code, rec.Body.String())
	}
}

// TestOpsGetMissingNotFoundHttp verifies a missing record maps to 404 instead
// of collapsing into a generic 500.
func TestOpsGetMissingNotFoundHttp(t *testing.T) {
	svc := newOpsService(nil)
	router := opsErrorTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/ops/records/missing-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing record, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// TestOpsCreateDuplicateConflictHttp verifies a duplicate create returns 409,
// not a 500.
func TestOpsCreateDuplicateConflictHttp(t *testing.T) {
	svc := newOpsService(nil)
	router := opsErrorTestRouter(svc)
	createOpsRecord(t, router, "op-dup")
	body := `{"id":"op-dup","subject":"daily check","owner":"alice","priority":"high","labels":{"site":"west"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate create should be 409, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// TestOpsTransitionConflictHttp verifies an illegal status transition returns
// 409 rather than a generic 500.
func TestOpsTransitionConflictHttp(t *testing.T) {
	svc := newOpsService(nil)
	router := opsErrorTestRouter(svc)
	createOpsRecord(t, router, "op-tx")
	req := httptest.NewRequest(http.MethodPost, "/api/ops/records/op-tx/transition", strings.NewReader(`{"target":"bogus"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("illegal transition should be 409, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// TestOpsCodeRecognizesNestedSentinels verifies an OpsError wrapping a
// sentinel resolves to the sentinel code, not the generic operation code.
func TestOpsCodeRecognizesNestedSentinels(t *testing.T) {
	err := wrapOps("create", "store.put", ErrOpsConflict)
	if code := opsCode(err); code != "conflict" {
		t.Fatalf("expected conflict, got %q", code)
	}
	err = wrapOps("get", "store.get", ErrOpsNotFound)
	if code := opsCode(err); code != "not_found" {
		t.Fatalf("expected not_found, got %q", code)
	}
}
