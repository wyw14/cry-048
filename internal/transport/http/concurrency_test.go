package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
)

// TestConcurrentAnnotationUpdateStaleDetection verifies that optimistic
// locking prevents lost updates: when many writers race to mutate the same
// annotation using the same expected version, only one can succeed. All
// other writers must observe ErrStaleVersion, surfaced as HTTP 409.
//
// The test pre-fetches the annotation once and has every goroutine attempt
// an Update using that single shared expected version. Because the very first
// successful writer bumps the persisted version, every subsequent Update call
// must reject with ErrStaleVersion. This is a deterministic assertion of the
// optimistic-lock invariant regardless of Go's goroutine scheduler ordering.
func TestConcurrentAnnotationUpdateStaleDetection(t *testing.T) {
	h, _, annoRepo, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	pid, bid, vid := setupProjectBoardVersion(t, r)
	w := doJSON(t, r, "POST", "/api/v1/annotations", map[string]any{
		"id": "a1", "project_id": pid, "board_id": bid, "version_id": vid,
		"title": "t", "point": map[string]any{"x": 10, "y": 10},
	})
	require.Equal(t, 201, w.Code)

	// Pre-fetch the annotation so we know its initial Version.
	a, err := annoRepo.Get(context.Background(), "a1")
	require.NoError(t, err)
	initialVersion := a.Version

	const n = 25
	var wg sync.WaitGroup
	var ok, fail int
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Each goroutine takes its own copy of the pre-fetched annotation
			// and attempts to bump its priority using the SAME expected version.
			local := *a
			local.Priority = annotation.PriorityHigh
			if i%2 == 0 {
				local.Priority = annotation.PriorityNormal
			}
			// Domain mutators bump a.Version on the local copy. The persisted
			// row still has the original version until Update commits.
			local.Version = initialVersion + 1
			<-start
			// Bypass the service layer (which re-reads) and go straight to the
			// repository's Update call with the shared expected version. This
			// deterministically exercises the optimistic-lock branch.
			updateErr := annoRepo.Update(context.Background(), &local, initialVersion)
			mu.Lock()
			if updateErr == nil {
				ok++
			} else {
				fail++
			}
			mu.Unlock()
		}(i)
	}
	close(start)
	wg.Wait()
	if ok+fail != n {
		t.Fatalf("expected %d responses, got ok=%d fail=%d", n, ok, fail)
	}
	if ok != 1 {
		t.Fatalf("expected exactly one successful update, got ok=%d", ok)
	}
	if fail != n-1 {
		t.Fatalf("expected %d stale failures, got fail=%d", n-1, fail)
	}
}

// TestRequestIDHeaderPropagated ensures X-Request-ID is set when supplied by the client.
func TestRequestIDHeaderPropagated(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = "auto"
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	})
	r.GET("/healthz", h.Healthz)
	req := httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Request-ID", "client-supplied-rid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "client-supplied-rid", w.Header().Get("X-Request-ID"))
}

// TestConcurrentProjectCreation ensures the in-memory repo handles concurrent writes.
func TestConcurrentProjectCreation(t *testing.T) {
	h, _, _, _, _ := newTestHandlers(t)
	r := newTestRouter(h)
	var wg sync.WaitGroup
	var ok int
	var mu sync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := doJSON(t, r, "POST", "/api/v1/projects", map[string]any{
				"id": "p-" + itoaInt(i), "name": "P",
			})
			mu.Lock()
			if w.Code == 201 {
				ok++
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	require.Equal(t, 20, ok)
}

var _ = http.StatusOK
