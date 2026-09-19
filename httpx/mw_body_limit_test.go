package httpx

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samuelsih/golib/assert"
)

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

type testReadCloser struct {
	io.Reader
	closed bool
}

func (c *testReadCloser) Close() error {
	c.closed = true
	return nil
}

func newBodyLimitReader(body string, limit int64) *bodyLimitReader {
	rc := &testReadCloser{Reader: strings.NewReader(body)}
	return &bodyLimitReader{Reader: rc, Closer: rc, limit: limit}
}

func testBodyRequest(t *testing.T, r *Router, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body)))

	return rec
}

func TestBodyLimitReaderRead(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		r := newBodyLimitReader("", 10)

		got, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, string(got), "")
	})

	t.Run("body within limit", func(t *testing.T) {
		r := newBodyLimitReader("hello", 10)

		got, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, string(got), "hello")
	})

	t.Run("body equal to limit", func(t *testing.T) {
		r := newBodyLimitReader("hello", 5)

		got, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, string(got), "hello")
	})

	t.Run("body over limit", func(t *testing.T) {
		r := newBodyLimitReader("hello world", 5)

		got, err := io.ReadAll(r)
		assert.Equal(t, string(got), "hello")

		maxErr := assert.ErrorAsType[*http.MaxBytesError](t, err)
		assert.Equal(t, maxErr.Limit, int64(5))
	})

	t.Run("zero limit rejects any body", func(t *testing.T) {
		r := newBodyLimitReader("a", 0)

		got, err := io.ReadAll(r)
		assert.Equal(t, string(got), "")

		maxErr := assert.ErrorAsType[*http.MaxBytesError](t, err)
		assert.Equal(t, maxErr.Limit, int64(0))
	})

	t.Run("small reads across the limit", func(t *testing.T) {
		r := newBodyLimitReader("abcdefgh", 5)
		buf := make([]byte, 3)

		n, err := r.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, n, 3)
		assert.Equal(t, string(buf[:n]), "abc")

		n, err = r.Read(buf)
		assert.Equal(t, n, 2)
		assert.Equal(t, string(buf[:n]), "de")

		maxErr := assert.ErrorAsType[*http.MaxBytesError](t, err)
		assert.Equal(t, maxErr.Limit, int64(5))
	})

	t.Run("over limit error is sticky", func(t *testing.T) {
		r := newBodyLimitReader("hello world", 5)

		_, first := io.ReadAll(r)

		_, second := r.Read(make([]byte, 10))
		assert.ErrorIs(t, second, first)
	})

	t.Run("underlying error within limit is sticky", func(t *testing.T) {
		want := errors.New("read failure")
		rc := &testReadCloser{Reader: errReader{err: want}}
		r := &bodyLimitReader{Reader: rc, Closer: rc, limit: 10}

		_, err := io.ReadAll(r)
		assert.ErrorIs(t, err, want)

		_, err = r.Read(make([]byte, 10))
		assert.ErrorIs(t, err, want)
	})

	t.Run("zero length buffer does not consume body", func(t *testing.T) {
		r := newBodyLimitReader("hello", 5)

		n, err := r.Read(nil)
		assert.NoError(t, err)
		assert.Equal(t, n, 0)

		got, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, string(got), "hello")
	})

	t.Run("zero length buffer returns sticky error", func(t *testing.T) {
		r := newBodyLimitReader("hello world", 5)

		_, first := io.ReadAll(r)

		_, second := r.Read(nil)
		assert.ErrorIs(t, second, first)
	})

	t.Run("close propagates to the underlying body", func(t *testing.T) {
		rc := &testReadCloser{Reader: strings.NewReader("hello")}
		r := &bodyLimitReader{Reader: rc, Closer: rc, limit: 10}

		assert.NoError(t, r.Close())
		assert.True(t, rc.closed)
	})
}

func TestMiddlewareMaxBytes(t *testing.T) {
	t.Run("wraps the request body and stores the limiter in context", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", strings.NewReader("hello"))

		handler := MiddlewareMaxBytes(10)(func(_ http.ResponseWriter, r *http.Request) error {
			limiter, ok := r.Context().Value(maxBytesCtxKey).(*bodyLimitReader)
			assert.True(t, ok)
			assert.Equal(t, limiter.limit, int64(10))
			assert.True(t, r.Body == limiter)

			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, string(got), "hello")

			return nil
		})

		assert.NoError(t, handler(httptest.NewRecorder(), req))
	})

	t.Run("rejects body over the limit", func(t *testing.T) {
		var got string

		handler := MiddlewareMaxBytes(4)(func(_ http.ResponseWriter, r *http.Request) error {
			body, err := io.ReadAll(r.Body)
			got = string(body)
			return err
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", strings.NewReader("hello world"))
		err := handler(httptest.NewRecorder(), req)

		assert.Equal(t, got, "hell")

		maxErr := assert.ErrorAsType[*http.MaxBytesError](t, err)
		assert.Equal(t, maxErr.Limit, int64(4))
	})

	t.Run("nested middleware tightens the limit", func(t *testing.T) {
		var got string

		inner := MiddlewareMaxBytes(4)(func(_ http.ResponseWriter, r *http.Request) error {
			body, err := io.ReadAll(r.Body)
			got = string(body)
			return err
		})
		handler := MiddlewareMaxBytes(64)(inner)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", strings.NewReader("hello world"))
		err := handler(httptest.NewRecorder(), req)

		assert.Equal(t, got, "hell")

		maxErr := assert.ErrorAsType[*http.MaxBytesError](t, err)
		assert.Equal(t, maxErr.Limit, int64(4))
	})

	t.Run("limit applies only when the body is read", func(t *testing.T) {
		handler := MiddlewareMaxBytes(1)(func(http.ResponseWriter, *http.Request) error {
			return nil
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", strings.NewReader("hello world"))
		assert.NoError(t, handler(httptest.NewRecorder(), req))
	})

	t.Run("closing the body closes the underlying body", func(t *testing.T) {
		rc := &testReadCloser{Reader: strings.NewReader("hello")}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", nil)
		req.Body = rc

		handler := MiddlewareMaxBytes(10)(func(_ http.ResponseWriter, r *http.Request) error {
			return r.Body.Close()
		})

		assert.NoError(t, handler(httptest.NewRecorder(), req))
		assert.True(t, rc.closed)
	})
}

func TestMiddlewareMaxBytesRouter(t *testing.T) {
	r := NewRouter()
	r.Post("/upload", func(w http.ResponseWriter, req *http.Request) error {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}

		_, err = w.Write(body)
		return err
	}, MiddlewareMaxBytes(5))

	t.Run("body within limit", func(t *testing.T) {
		rec := testBodyRequest(t, r, http.MethodPost, "/upload", "hello")
		assert.Equal(t, rec.Code, http.StatusOK)
		assert.Equal(t, rec.Body.String(), "hello")
	})

	t.Run("body over limit", func(t *testing.T) {
		rec := testBodyRequest(t, r, http.MethodPost, "/upload", "hello world")
		assert.Equal(t, rec.Code, http.StatusRequestEntityTooLarge)
		assert.Equal(t, rec.Body.String(), http.StatusText(http.StatusRequestEntityTooLarge)+"\n")
	})
}
