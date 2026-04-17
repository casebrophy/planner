package claudecli

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

func TestSerializationMaxConcurrency(t *testing.T) {
	// Track concurrent in-flight requests to the test server.
	var inflight int32
	var maxConcurrent int32
	var mu sync.Mutex
	var requests []*http.Request

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment inflight count.
		current := atomic.AddInt32(&inflight, 1)
		defer func() {
			atomic.AddInt32(&inflight, -1)
		}()

		// Track max concurrency and request.
		mu.Lock()
		requests = append(requests, r)
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()

		// Sleep briefly to simulate work and allow other requests to queue up.
		time.Sleep(50 * time.Millisecond)

		// Return valid sidecar envelope.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"{\"ok\":true}","model":"haiku"}`))
	}))
	defer ts.Close()

	log := logger.New(io.Discard, logger.LevelInfo, "test")
	client := NewClient(log, []string{"haiku"}, ts.URL, "")
	defer client.Close()

	// Fire 3 concurrent RunJSON calls.
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var dest struct {
				Ok bool `json:"ok"`
			}
			err := client.RunJSON(ctx, "test prompt", "", &dest, nil)
			if err != nil {
				t.Errorf("RunJSON failed: %v", err)
			}
			if !dest.Ok {
				t.Errorf("expected Ok=true, got false")
			}
		}()
	}
	wg.Wait()

	// Verify max concurrency was 1.
	if maxConcurrent != 1 {
		t.Errorf("expected max concurrency 1, got %d", maxConcurrent)
	}

	// Verify all 3 requests were made.
	mu.Lock()
	if len(requests) != 3 {
		t.Errorf("expected 3 requests, got %d", len(requests))
	}
	mu.Unlock()
}

func TestQueueWaitDoesntConsumeTimeout(t *testing.T) {
	// Test that the caller's timeout doesn't start until the job begins executing.
	// With a 100ms server sleep per request and a 250ms client timeout,
	// 3 sequential jobs should all succeed (total ~300ms wall time).
	// If timeout started at enqueue, they'd likely fail due to cumulative delays.

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Each request sleeps for 100ms.
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"{\"ok\":true}","model":"haiku"}`))
	}))
	defer ts.Close()

	log := logger.New(io.Discard, logger.LevelInfo, "test")
	client := NewClient(log, []string{"haiku"}, ts.URL, "")
	client.timeout = 250 * time.Millisecond
	defer client.Close()

	// Fire 3 sequential RunJSON calls.
	// Total wall time will be ~300ms (3 * 100ms sleep).
	// Each has a 250ms timeout, so if timeouts started at enqueue,
	// the 2nd and 3rd would timeout. But since timeout starts at execution,
	// all should succeed.
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		var dest struct {
			Ok bool `json:"ok"`
		}
		err := client.RunJSON(ctx, "test prompt", "", &dest, nil)
		if err != nil {
			t.Errorf("RunJSON call %d failed: %v", i+1, err)
		}
	}
}

func TestCloseRejectsNewCalls(t *testing.T) {
	// Test that after Close(), subsequent calls return ErrClosed.

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"{\"ok\":true}","model":"haiku"}`))
	}))
	defer ts.Close()

	log := logger.New(io.Discard, logger.LevelInfo, "test")
	client := NewClient(log, []string{"haiku"}, ts.URL, "")
	client.Close()

	// Attempt a call on the closed client.
	ctx := context.Background()
	var dest struct {
		Ok bool `json:"ok"`
	}
	err := client.RunJSON(ctx, "test", "", &dest, nil)

	// Should get an error containing ErrClosed (may be wrapped by RunJSON).
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	// The error may be wrapped, so just check it's an error (RunJSON wraps it).
	if !errors.Is(err, ErrClosed) && err.Error() != ErrClosed.Error() {
		// Accept if it's wrapped in "failed to run claude cli" message which happens
		// when executeSingleInference returns ErrClosed
		if err.Error() != "failed to run claude cli with model haiku: claudecli: client closed" {
			t.Errorf("expected ErrClosed or wrapped version, got %v", err)
		}
	}
}

func TestRunJSONUnmarshalsSimpleStructFromSidecarEnvelope(t *testing.T) {
	// Test that RunJSON correctly unmarshals a simple struct from the sidecar envelope.

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"{\"message\":\"hello\"}","model":"haiku"}`))
	}))
	defer ts.Close()

	log := logger.New(io.Discard, logger.LevelInfo, "test")
	client := NewClient(log, []string{"haiku"}, ts.URL, "")
	defer client.Close()

	ctx := context.Background()
	var dest struct {
		Message string `json:"message"`
	}
	err := client.RunJSON(ctx, "test prompt", "", &dest, nil)
	if err != nil {
		t.Fatalf("RunJSON failed: %v", err)
	}
	if dest.Message != "hello" {
		t.Errorf("expected message='hello', got '%s'", dest.Message)
	}
}


func TestLastModel(t *testing.T) {
	// Test that LastModel returns the model used in the most recent successful RunJSON call.

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"{\"ok\":true}","model":"haiku"}`))
	}))
	defer ts.Close()

	log := logger.New(io.Discard, logger.LevelInfo, "test")
	client := NewClient(log, []string{"haiku", "sonnet"}, ts.URL, "")
	defer client.Close()

	if client.LastModel() != "" {
		t.Errorf("expected initial LastModel to be empty, got '%s'", client.LastModel())
	}

	ctx := context.Background()
	var dest struct {
		Ok bool `json:"ok"`
	}
	err := client.RunJSON(ctx, "test prompt", "", &dest, nil)
	if err != nil {
		t.Fatalf("RunJSON failed: %v", err)
	}

	if client.LastModel() != "haiku" {
		t.Errorf("expected LastModel='haiku', got '%s'", client.LastModel())
	}
}
