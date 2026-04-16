package ollamaclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestFIFOOrder verifies that concurrent Do calls are processed strictly
// one-at-a-time and in the exact order they were enqueued.
func TestFIFOOrder(t *testing.T) {
	t.Parallel()

	const n = 5

	var (
		mu         sync.Mutex
		inFlight   int
		maxObserve int
		order      []string
	)

	// Gate the server so nothing executes until all jobs are enqueued.
	gate := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-gate // hold until test releases

		mu.Lock()
		inFlight++
		if inFlight > maxObserve {
			maxObserve = inFlight
		}
		if inFlight > 1 {
			mu.Unlock()
			t.Errorf("concurrent in-flight requests observed: %d", inFlight)
			http.Error(w, "concurrency violation", http.StatusInternalServerError)
			return
		}
		mu.Unlock()

		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		// Simulate work so a FIFO violation would be observable.
		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		order = append(order, string(body))
		inFlight--
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := New(Config{BaseURL: srv.URL, QueueSize: n})
	defer client.Close()

	// Enqueue n jobs sequentially from a single producer goroutine; each Do
	// blocks the producer until its result arrives, so we spawn a goroutine
	// per Do. We serialize enqueue ordering by waiting for the queue-send to
	// have been observed before launching the next.
	results := make([]error, n)
	var wg sync.WaitGroup

	// Launch per-job goroutines one at a time. We sleep between launches to
	// give each goroutine time to reach `queue <- j` before the next one
	// starts. Since QueueSize == n, all enqueues are non-blocking and quick.
	for i := 0; i < n; i++ {
		wg.Add(1)
		idx := i
		tag := fmt.Sprintf("job-%d", idx)
		go func() {
			defer wg.Done()
			_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
				return http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/x",
					newStringReader(tag))
			})
			results[idx] = err
		}()
		time.Sleep(5 * time.Millisecond)
	}

	// All n jobs should now be in the FIFO queue (or one in-flight, held at
	// the gate). Release the gate and let them flow.
	close(gate)
	wg.Wait()

	for i, err := range results {
		if err != nil {
			t.Fatalf("result %d: %v", i, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if maxObserve > 1 {
		t.Fatalf("observed concurrent requests: max=%d", maxObserve)
	}
	if len(order) != n {
		t.Fatalf("order count: got %d want %d", len(order), n)
	}
	for i, got := range order {
		want := fmt.Sprintf("job-%d", i)
		if got != want {
			t.Fatalf("order[%d]: got %q want %q (full=%v)", i, got, want, order)
		}
	}
}

// TestQueuedJobSurvivesCallerCtxCancel verifies that once a job is enqueued,
// cancelling the caller's ctx does NOT cancel the job. The server still sees
// the request and returns a proper result.
func TestQueuedJobSurvivesCallerCtxCancel(t *testing.T) {
	t.Parallel()

	// A signal the server uses to block job A until we release it.
	release := make(chan struct{})
	var bSeen atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/a":
			<-release
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"a":true}`))
		case "/b":
			bSeen.Store(true)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"b":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := New(Config{BaseURL: srv.URL})
	defer client.Close()

	// Submit A (will block on server).
	aDone := make(chan error, 1)
	go func() {
		_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/a", nil)
		})
		aDone <- err
	}()

	// Wait until A is actually dequeued and in-flight; we detect this by
	// watching for the queue to become empty. Simplest proxy: small sleep.
	time.Sleep(30 * time.Millisecond)

	// Submit B with a short-lived ctx. Because A is blocking the worker,
	// B sits in the queue. Its ctx will cancel, but Do should already have
	// enqueued it and thus NOT return ctx.Err().
	bCtx, bCancel := context.WithCancel(context.Background())
	bDone := make(chan struct {
		body []byte
		err  error
	}, 1)
	go func() {
		body, err := client.Do(bCtx, func(ctx context.Context) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/b", nil)
		})
		bDone <- struct {
			body []byte
			err  error
		}{body, err}
	}()

	// Give B time to enqueue (queue has room; A is in-flight).
	time.Sleep(30 * time.Millisecond)
	bCancel() // cancel AFTER B is enqueued

	// Release A so the worker can process B next.
	close(release)

	select {
	case err := <-aDone:
		if err != nil {
			t.Fatalf("A failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("A timed out")
	}

	select {
	case res := <-bDone:
		if res.err != nil {
			t.Fatalf("B returned error instead of completing: %v", res.err)
		}
		if !bSeen.Load() {
			t.Fatal("server never saw B's request")
		}
		if string(res.body) != `{"b":true}` {
			t.Fatalf("B body: %q", string(res.body))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("B timed out")
	}
}

// TestEnqueueRespectsCtxCancel verifies that when the queue is full and the
// caller's ctx is already cancelled, Do returns ctx.Err() without reaching
// the server.
func TestEnqueueRespectsCtxCancel(t *testing.T) {
	t.Parallel()

	var serverHits atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHits.Add(1)
		<-release
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := New(Config{BaseURL: srv.URL, QueueSize: 1})
	defer func() {
		close(release)
		client.Close()
	}()

	// Submit a blocking job to occupy the worker.
	blockerDone := make(chan error, 1)
	go func() {
		_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/", nil)
		})
		blockerDone <- err
	}()
	time.Sleep(30 * time.Millisecond)

	// Submit a second job to fill the queue (QueueSize=1).
	fillerDone := make(chan error, 1)
	go func() {
		_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/", nil)
		})
		fillerDone <- err
	}()
	time.Sleep(30 * time.Millisecond)

	// Now a third Do with an already-cancelled ctx should return ctx.Err()
	// without reaching the server.
	cctx, cancel := context.WithCancel(context.Background())
	cancel()

	hitsBefore := serverHits.Load()
	_, err := client.Do(cctx, func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/", nil)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do: got %v, want context.Canceled", err)
	}

	// The third job must not have reached the server. (serverHits can include
	// the blocker if it was dispatched already; we just assert no increase
	// while the third request was rejected at the enqueue gate.)
	if got := serverHits.Load(); got != hitsBefore {
		t.Fatalf("server hit count changed: before=%d after=%d", hitsBefore, got)
	}
}

// TestCloseReturnsErrClosed verifies that after Close, Do returns ErrClosed.
func TestCloseReturnsErrClosed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := New(Config{BaseURL: srv.URL})
	client.Close()

	_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/", nil)
	})
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Do after Close: got %v, want ErrClosed", err)
	}
}

// TestCloseDrainsPending verifies that Close waits for the queue to drain.
func TestCloseDrainsPending(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := New(Config{BaseURL: srv.URL, QueueSize: 8})

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
				return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/", nil)
			})
			if err != nil {
				t.Errorf("Do: %v", err)
			}
		}()
	}

	// Let all 3 enqueue before Close.
	time.Sleep(10 * time.Millisecond)
	client.Close()
	wg.Wait()

	if got := hits.Load(); got != 3 {
		t.Fatalf("server hits after drain: got %d want 3", got)
	}
}

// TestEmbedAndGenerate round-trips the typed helpers against a canned server.
func TestEmbedAndGenerate(t *testing.T) {
	t.Parallel()

	type embedResp struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	type genResp struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	wantEmb := [][]float32{{0.1, 0.2, 0.3}}
	wantGen := `{"summary":"hi"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/embed":
			_ = json.NewEncoder(w).Encode(embedResp{Embeddings: wantEmb})
		case "/api/generate":
			_ = json.NewEncoder(w).Encode(genResp{Response: wantGen, Done: true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := New(Config{BaseURL: srv.URL})
	defer client.Close()

	embs, err := client.Embed(context.Background(), "nomic", []string{"hello"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(embs) != 1 || embs[0][0] != 0.1 {
		t.Fatalf("Embed result mismatch: %v", embs)
	}

	out, err := client.Generate(context.Background(), "llama3", "prompt")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if out != wantGen {
		t.Fatalf("Generate: got %q want %q", out, wantGen)
	}
}

// newStringReader is a tiny helper because we don't want to pull in strings
// or bytes just for one call site.
func newStringReader(s string) io.Reader {
	return &stringReader{s: s}
}

type stringReader struct {
	s string
	i int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}
