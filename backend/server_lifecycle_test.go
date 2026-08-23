package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

func shutdownCycle() int {
	server := &http.Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})}
	done := make(chan struct{})
	go func() {
		_ = serveHTTP(server)
		close(done)
	}()
	// Give the signal handler time to register before we interrupt the process.
	time.Sleep(300 * time.Millisecond)
	_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		return -1
	}
	time.Sleep(200 * time.Millisecond)
	return runtime.NumGoroutine()
}

// TestServeHTTPLeaksNoGoroutine verifies a graceful shutdown drains the
// ListenAndServe goroutine: two consecutive shutdown cycles must not grow the
// goroutine count.
func TestServeHTTPLeaksNoGoroutine(t *testing.T) {
	first := shutdownCycle()
	if first < 0 {
		t.Fatal("serveHTTP did not return after SIGTERM")
	}
	time.Sleep(100 * time.Millisecond)
	second := shutdownCycle()
	if second < 0 {
		t.Fatal("serveHTTP did not return after SIGTERM")
	}
	if second > first {
		t.Fatalf("goroutine count grew across shutdown cycles: first=%d second=%d", first, second)
	}
}

// TestLatencyHeaderPresent verifies every response carries the latency header.
// The check runs through a real HTTP server because headers set after the
// response has been written are dropped by net/http.
func TestLatencyHeaderPresent(t *testing.T) {
	server := newEnterpriseServer(":0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	ts := httptest.NewServer(server.Handler)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Operations-Latency-Ms") == "" {
		t.Fatal("expected X-Operations-Latency-Ms header on the response")
	}
}

// TestRequestIDHeaderPresent verifies responses echo the generated request id.
func TestRequestIDHeaderPresent(t *testing.T) {
	handler := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header on the response")
	}
}

// TestParallelRequestIDs verifies request id generation is safe under
// concurrent traffic.
func TestParallelRequestIDs(t *testing.T) {
	handler := requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				req := httptest.NewRequest(http.MethodGet, "/x", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
			}
		}()
	}
	wg.Wait()
}
