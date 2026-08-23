package api

import (
	"food-lab-chain-service/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func routeForTest() http.Handler {
	return NewRouter(store.New(), fstest.MapFS{"index.html": {Data: []byte("specimens")}, "app.js": {Data: []byte("fetch('/api/specimens')")}})
}
func TestReadRoutes(t *testing.T) {
	server := httptest.NewServer(routeForTest())
	defer server.Close()
	for _, path := range []string{"/healthz", "/api/specimens", "/", "/app.js"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
func TestHandoffPaths(t *testing.T) {
	server := httptest.NewServer(routeForTest())
	defer server.Close()
	bad, _ := http.Post(server.URL+"/api/specimens/spec-01/handoff", "application/json", strings.NewReader(`{"to":""}`))
	if bad.StatusCode != 400 {
		t.Fatalf("bad status: %d", bad.StatusCode)
	}
	bad.Body.Close()
	good, _ := http.Post(server.URL+"/api/specimens/spec-01/handoff", "application/json", strings.NewReader(`{"to":"Microbiology Bench"}`))
	if good.StatusCode != 200 {
		t.Fatalf("good status: %d", good.StatusCode)
	}
	good.Body.Close()
	sealed, _ := http.Post(server.URL+"/api/specimens/spec-02/handoff", "application/json", strings.NewReader(`{"to":"Review Bench"}`))
	if sealed.StatusCode != 409 {
		t.Fatalf("sealed status: %d", sealed.StatusCode)
	}
	sealed.Body.Close()
}
