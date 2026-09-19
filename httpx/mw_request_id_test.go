package httpx

import (
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/samuelsih/golib/assert"
)

func requestIDTestRouter() *Router {
	router := NewRouter()
	router.Use(MiddlewareRequestID())

	router.Get("/", func(w http.ResponseWriter, r *http.Request) error {
		_, err := io.WriteString(w, RequestID(r)) //nolint:gosec // the ID is generated server-side.
		return err
	})

	return router
}

func serveRequestID(t *testing.T, router *Router, header string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	if header != "" {
		req.Header.Set(RequestIDHeader, header)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func TestGenerateRequestID(t *testing.T) {
	before := uint64(time.Now().UnixNano())
	id := generateRequestID()
	after := uint64(time.Now().UnixNano())

	decoded, err := base64.RawURLEncoding.DecodeString(id)
	assert.NoError(t, err)
	assert.Equal(t, len(decoded), 16)

	timestamp := binary.BigEndian.Uint64(decoded[:8])
	assert.True(t, timestamp >= before && timestamp <= after)

	assert.NotEqual(t, id, generateRequestID())
}

func TestMiddlewareRequestIDGenerates(t *testing.T) {
	router := requestIDTestRouter()

	rec := serveRequestID(t, router, "")
	assert.Equal(t, rec.Code, http.StatusOK)

	id := rec.Body.String()
	assert.Equal(t, rec.Header().Get(RequestIDHeader), id)
	assert.Equal(t, len(id), 22)

	decoded, err := base64.RawURLEncoding.DecodeString(id)
	assert.NoError(t, err)
	assert.Equal(t, len(decoded), 16)
}

func TestMiddlewareRequestIDReuses(t *testing.T) {
	router := requestIDTestRouter()

	valid := []string{
		strings.Repeat("a", requestIDMinLength),
		strings.Repeat("b", requestIDMaxLength),
	}

	for _, incoming := range valid {
		rec := serveRequestID(t, router, incoming)
		assert.Equal(t, rec.Body.String(), incoming)
		assert.Equal(t, rec.Header().Get(RequestIDHeader), incoming)
	}
}

func TestMiddlewareRequestIDReplacesInvalid(t *testing.T) {
	router := requestIDTestRouter()

	invalid := []string{
		"short",
		strings.Repeat("a", requestIDMinLength-1),
		strings.Repeat("b", requestIDMaxLength+1),
	}

	for _, incoming := range invalid {
		rec := serveRequestID(t, router, incoming)
		assert.NotEqual(t, rec.Body.String(), incoming)
		assert.Equal(t, rec.Header().Get(RequestIDHeader), rec.Body.String())
	}
}

func TestMiddlewareRequestIDUnique(t *testing.T) {
	router := requestIDTestRouter()

	first := serveRequestID(t, router, "").Body.String()
	second := serveRequestID(t, router, "").Body.String()

	assert.NotEqual(t, first, second)
}

func TestRequestIDWithoutMiddleware(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	assert.Equal(t, RequestID(req), "")
}
