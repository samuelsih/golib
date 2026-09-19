package httpx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/samuelsih/golib/assert"
)

const (
	never    = 24 * time.Hour
	infinite = never

	// statusClientClosedRequest is the non-standard nginx status code for a
	// request whose client disconnected before a response was written.
	statusClientClosedRequest = 499
)

func timeoutChain(timeouts ...time.Duration) Middleware {
	return func(next Handler) Handler {
		for _, d := range slices.Backward(timeouts) {
			next = MiddlewareTimeout(d)(next)
		}
		return next
	}
}

// handler simulates work of the duration in the work_size query parameter.
// It writes 200 on success; on timeout or client disconnect it returns without
// writing a response and lets writeError turn the middleware error into 504/499.
func handler(w http.ResponseWriter, r *http.Request) error {
	work, err := time.ParseDuration(r.URL.Query().Get("work_size"))
	if err != nil {
		return err
	}

	select {
	case <-time.After(work):
		w.WriteHeader(http.StatusOK)
	case <-r.Context().Done():
	}

	return nil
}

func TestDisconnectAfterOriginalDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// 30s timeout overridden with 60s timeout
		r := NewRouter()
		r.Use(timeoutChain(30*time.Second, 60*time.Second))
		r.Get("/messages", handler)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// Client will disconnect after 45s
		go func() {
			time.Sleep(45 * time.Second)
			cancel()
		}()

		// request takes 50s normally (within the timeout)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/messages?work_size=50s", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		// We expect the "client disconnected" status
		assert.Equal(t, rec.Code, statusClientClosedRequest)
	})
}

func TestTimeoutTable(t *testing.T) {
	tests := []struct {
		name             string
		timeouts         []time.Duration // the last entry wins (effective timeout)
		workSize         time.Duration
		clientDisconnect time.Duration
		wantStatus       int
	}{
		{
			name:             "completes within the timeout",
			timeouts:         []time.Duration{30 * time.Second},
			workSize:         10 * time.Second,
			clientDisconnect: never,
			wantStatus:       http.StatusOK,
		},
		{
			name:             "times out when work exceeds it",
			timeouts:         []time.Duration{30 * time.Second},
			workSize:         40 * time.Second,
			clientDisconnect: never,
			wantStatus:       http.StatusGatewayTimeout,
		},
		{
			name:             "a later, shorter Timeout wins",
			timeouts:         []time.Duration{60 * time.Second, 30 * time.Second},
			workSize:         40 * time.Second,
			clientDisconnect: never,
			wantStatus:       http.StatusGatewayTimeout,
		},
		{
			name:             "a later, longer Timeout wins",
			timeouts:         []time.Duration{5 * time.Second, 30 * time.Second},
			workSize:         10 * time.Second,
			clientDisconnect: never,
			wantStatus:       http.StatusOK,
		},
		{
			name:             "disconnect before the timeout cancels",
			timeouts:         []time.Duration{30 * time.Second},
			workSize:         infinite,
			clientDisconnect: 10 * time.Second,
			wantStatus:       statusClientClosedRequest,
		},
		{
			name:             "disconnect after the original deadline still cancels",
			timeouts:         []time.Duration{5 * time.Second, 30 * time.Second},
			workSize:         infinite,
			clientDisconnect: 10 * time.Second,
			wantStatus:       statusClientClosedRequest,
		},
		{
			name:             "disconnect after the timeout has fired stays a timeout",
			timeouts:         []time.Duration{5 * time.Second, 30 * time.Second},
			workSize:         infinite,
			clientDisconnect: 40 * time.Second,
			wantStatus:       http.StatusGatewayTimeout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := NewRouter()
				r.Use(timeoutChain(tc.timeouts...))
				r.Get("/messages", handler)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if tc.clientDisconnect != never {
					time.AfterFunc(tc.clientDisconnect, cancel)
				}

				target := fmt.Sprintf("/messages?work_size=%s", tc.workSize)
				req := httptest.NewRequestWithContext(ctx, http.MethodGet, target, nil)
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, req)

				assert.Equal(t, rec.Code, tc.wantStatus)
			})
		})
	}
}

func TestDeadlineReported(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()

		expect := func(expected time.Duration, ctx context.Context) {
			t.Helper()
			d, ok := ctx.Deadline()
			assert.True(t, ok)
			assert.Equal(t, time.Until(d), expected)
		}

		// A real parent deadline is reported as-is.
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		expect(30*time.Second, ctx)

		// A shorter dynamic timeout under that parent wins.
		ctx, cancelTimeout, reset := withTimeout(ctx, 20*time.Second)
		defer cancelTimeout()
		expect(20*time.Second, ctx)

		// Resetting moves the reported deadline while it stays under the parent.
		reset(25 * time.Second)
		expect(25*time.Second, ctx)

		// Resetting past the parent is capped at the parent's 30s.
		reset(40 * time.Second)
		expect(30*time.Second, ctx)
	})
}
