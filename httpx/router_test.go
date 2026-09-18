package httpx

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samuelsih/golib/assert"
)

func testRequest(t *testing.T, r *Router, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, nil))

	return rec
}

func testHandler(body string) Handler {
	return func(w http.ResponseWriter, _ *http.Request) error {
		_, err := io.WriteString(w, body)
		return err
	}
}

func testErrorHandler(err error) Handler {
	return func(http.ResponseWriter, *http.Request) error {
		return err
	}
}

func testTracer(calls *[]string, name string) Middleware {
	return func(next Handler) Handler {
		return func(w http.ResponseWriter, req *http.Request) error {
			*calls = append(*calls, name)
			return next(w, req)
		}
	}
}

func TestRouterHandleMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		register func(*Router, string, Handler, ...Middleware)
	}{
		{name: "get", method: http.MethodGet, register: (*Router).Get},
		{name: "post", method: http.MethodPost, register: (*Router).Post},
		{name: "put", method: http.MethodPut, register: (*Router).Put},
		{name: "delete", method: http.MethodDelete, register: (*Router).Delete},
		{name: "head", method: http.MethodHead, register: (*Router).Head},
		{name: "options", method: http.MethodOptions, register: (*Router).Options},
		{name: "trace", method: http.MethodTrace, register: (*Router).Trace},
		{name: "query", method: "QUERY", register: (*Router).Query},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			tt.register(r, "/things", testHandler("ok"))

			rec := testRequest(t, r, tt.method, "/things")
			assert.Equal(t, rec.Code, http.StatusOK)
			assert.Equal(t, rec.Body.String(), "ok")

			rec = testRequest(t, r, http.MethodPatch, "/things")
			assert.Equal(t, rec.Code, http.StatusMethodNotAllowed)
			assert.Equal(t, rec.Body.String(), http.StatusText(http.StatusMethodNotAllowed)+"\n")
		})
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	t.Run("registered path with wrong method", func(t *testing.T) {
		r := NewRouter()
		r.Get("/things", testHandler("get"))

		rec := testRequest(t, r, http.MethodPost, "/things")
		assert.Equal(t, rec.Code, http.StatusMethodNotAllowed)
		assert.Equal(t, rec.Body.String(), http.StatusText(http.StatusMethodNotAllowed)+"\n")
	})

	t.Run("multiple methods on the same path", func(t *testing.T) {
		r := NewRouter()
		r.Get("/things", testHandler("get"))
		r.Post("/things", testHandler("post"))

		assert.Equal(t, testRequest(t, r, http.MethodGet, "/things").Code, http.StatusOK)
		assert.Equal(t, testRequest(t, r, http.MethodPost, "/things").Code, http.StatusOK)
		assert.Equal(t, testRequest(t, r, http.MethodPut, "/things").Code, http.StatusMethodNotAllowed)
	})

	t.Run("wrong method on root route", func(t *testing.T) {
		r := NewRouter()
		r.Get("/", testHandler("root"))

		assert.Equal(t, testRequest(t, r, http.MethodPost, "/").Code, http.StatusMethodNotAllowed)
		assert.Equal(t, testRequest(t, r, http.MethodGet, "/missing").Code, http.StatusNotFound)
	})

	t.Run("wrong method on group route", func(t *testing.T) {
		r := NewRouter()
		r.GroupPrefix("/api", func(g *Router) {
			g.Get("/users", testHandler("users"))
		})

		rec := testRequest(t, r, http.MethodDelete, "/api/users")
		assert.Equal(t, rec.Code, http.StatusMethodNotAllowed)
	})

	t.Run("error handler receives method not allowed", func(t *testing.T) {
		r := NewRouter()
		r.RegisterErrorHandler(func(w http.ResponseWriter, _ *http.Request, err error) HandleState {
			assert.ErrorIs(t, err, ErrMethodNotAllowed)
			w.WriteHeader(http.StatusTeapot)
			return HandleStop
		})
		r.Get("/things", testHandler("get"))

		rec := testRequest(t, r, http.MethodPost, "/things")
		assert.Equal(t, rec.Code, http.StatusTeapot)
	})
}

func TestRouterWalk(t *testing.T) {
	r := NewRouter()
	r.Get("/", testHandler("root"))
	r.Post("/things", testHandler("post"))
	r.Query("/search", testHandler("query"))
	r.GroupPrefix("/api", func(g *Router) {
		g.Get("/users", testHandler("users"))
		g.Delete("/users/{id}", testHandler("delete"))
	})

	var got []Endpoint
	r.Walk(func(e Endpoint) {
		got = append(got, e)
	})

	want := []Endpoint{
		{Method: http.MethodGet, PatternURL: "/"},
		{Method: http.MethodPost, PatternURL: "/things"},
		{Method: "QUERY", PatternURL: "/search"},
		{Method: http.MethodGet, PatternURL: "/api/users"},
		{Method: http.MethodDelete, PatternURL: "/api/users/{id}"},
	}
	assert.Equal(t, got, want)
}

func TestRouterWalkNil(_ *testing.T) {
	r := NewRouter()
	r.Get("/", testHandler("root"))

	r.Walk(nil)
}

func TestRouterWalkSharedAcrossGroups(t *testing.T) {
	r := NewRouter()
	r.Get("/root", testHandler("root"))

	var got []Endpoint
	r.GroupPrefix("/api", func(g *Router) {
		g.Post("/users", testHandler("users"))
		g.Walk(func(e Endpoint) {
			got = append(got, e)
		})
	})

	want := []Endpoint{
		{Method: http.MethodGet, PatternURL: "/root"},
		{Method: http.MethodPost, PatternURL: "/api/users"},
	}
	assert.Equal(t, got, want)
}

func TestRouterNotFound(t *testing.T) {
	t.Run("unmatched route", func(t *testing.T) {
		r := NewRouter()

		rec := testRequest(t, r, http.MethodGet, "/missing")
		assert.Equal(t, rec.Code, http.StatusNotFound)
		assert.Equal(t, rec.Body.String(), http.StatusText(http.StatusNotFound)+"\n")
	})

	t.Run("root route is not catch-all", func(t *testing.T) {
		r := NewRouter()
		r.Get("/", testHandler("root"))

		rec := testRequest(t, r, http.MethodGet, "/")
		assert.Equal(t, rec.Code, http.StatusOK)
		assert.Equal(t, rec.Body.String(), "root")

		rec = testRequest(t, r, http.MethodGet, "/missing")
		assert.Equal(t, rec.Code, http.StatusNotFound)
	})
}

func TestRouterPathPrefix(t *testing.T) {
	r := NewRouter(WithPathPrefix("/api/v1"))
	r.Get("/users", testHandler("users"))

	assert.Equal(t, testRequest(t, r, http.MethodGet, "/api/v1/users").Code, http.StatusOK)
	assert.Equal(t, testRequest(t, r, http.MethodGet, "/users").Code, http.StatusNotFound)
}

func TestRouterGroup(t *testing.T) {
	r := NewRouter()
	r.Get("/root", testHandler("root"))
	r.Group(func(g *Router) {
		g.Get("/plain", testHandler("plain"))
	})
	r.GroupPrefix("/api", func(g *Router) {
		g.Get("/users", testHandler("users"))
		g.GroupPrefix("/v2", func(g *Router) {
			g.Get("/users", testHandler("v2-users"))
		})
	})

	tests := []struct {
		target string
		want   string
	}{
		{target: "/root", want: "root"},
		{target: "/plain", want: "plain"},
		{target: "/api/users", want: "users"},
		{target: "/api/v2/users", want: "v2-users"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			rec := testRequest(t, r, http.MethodGet, tt.target)
			assert.Equal(t, rec.Code, http.StatusOK)
			assert.Equal(t, rec.Body.String(), tt.want)
		})
	}

	assert.Equal(t, testRequest(t, r, http.MethodGet, "/v2/users").Code, http.StatusNotFound)
}

func TestRouterMiddlewareOrder(t *testing.T) {
	var calls []string

	r := NewRouter()
	r.Use(testTracer(&calls, "chain-1"), testTracer(&calls, "chain-2"))
	r.Get("/route", testHandler("ok"), testTracer(&calls, "route"))

	rec := testRequest(t, r, http.MethodGet, "/route")
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, calls, []string{"chain-1", "chain-2", "route"})
}

func TestRouterMiddlewareScope(t *testing.T) {
	var calls []string

	r := NewRouter()
	r.Use(testTracer(&calls, "before"))
	r.GroupPrefix("/api", func(g *Router) {
		g.Get("/in", testHandler("in"))
	})
	r.Use(testTracer(&calls, "after"))
	r.Get("/out", testHandler("out"))

	rec := testRequest(t, r, http.MethodGet, "/api/in")
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, calls, []string{"before"})

	calls = nil

	rec = testRequest(t, r, http.MethodGet, "/out")
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, calls, []string{"before", "after"})
}

func TestRouterRouteMiddleware(t *testing.T) {
	var calls []string

	r := NewRouter()
	r.Get("/with", testHandler("with"), testTracer(&calls, "only"))
	r.Get("/without", testHandler("without"))

	rec := testRequest(t, r, http.MethodGet, "/with")
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, calls, []string{"only"})

	calls = nil

	rec = testRequest(t, r, http.MethodGet, "/without")
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Nil(t, calls)
}

func TestRouterMiddlewareError(t *testing.T) {
	r := NewRouter()
	r.Use(func(Handler) Handler {
		return func(http.ResponseWriter, *http.Request) error {
			return ErrRouteNotFound
		}
	})
	r.Get("/route", testHandler("ok"))

	rec := testRequest(t, r, http.MethodGet, "/route")
	assert.Equal(t, rec.Code, http.StatusNotFound)
}

func TestRouterErrorHandlers(t *testing.T) {
	t.Run("stop prevents default handling", func(t *testing.T) {
		r := NewRouter()
		r.RegisterErrorHandler(func(w http.ResponseWriter, _ *http.Request, err error) HandleState {
			http.Error(w, "custom: "+err.Error(), http.StatusTeapot)
			return HandleStop
		})
		r.Get("/route", testErrorHandler(errors.New("boom")))

		rec := testRequest(t, r, http.MethodGet, "/route")
		assert.Equal(t, rec.Code, http.StatusTeapot)
		assert.Equal(t, rec.Body.String(), "custom: boom\n")
	})

	t.Run("continue falls through to default handling", func(t *testing.T) {
		r := NewRouter()
		r.RegisterErrorHandler(func(http.ResponseWriter, *http.Request, error) HandleState {
			return HandleContinue
		})
		r.Get("/route", testErrorHandler(errors.New("boom")))

		rec := testRequest(t, r, http.MethodGet, "/route")
		assert.Equal(t, rec.Code, http.StatusInternalServerError)
		assert.Equal(t, rec.Body.String(), http.StatusText(http.StatusInternalServerError)+"\n")
	})

	t.Run("handlers run in registration order", func(t *testing.T) {
		var calls []string

		r := NewRouter()
		r.RegisterErrorHandler(func(http.ResponseWriter, *http.Request, error) HandleState {
			calls = append(calls, "first")
			return HandleContinue
		})
		r.RegisterErrorHandler(func(w http.ResponseWriter, _ *http.Request, _ error) HandleState {
			calls = append(calls, "second")
			w.WriteHeader(http.StatusTeapot)
			return HandleStop
		})
		r.Get("/route", testErrorHandler(errors.New("boom")))

		rec := testRequest(t, r, http.MethodGet, "/route")
		assert.Equal(t, rec.Code, http.StatusTeapot)
		assert.Equal(t, calls, []string{"first", "second"})
	})

	t.Run("nil handler is ignored", func(t *testing.T) {
		r := NewRouter()
		r.RegisterErrorHandler(nil)
		r.Get("/route", testErrorHandler(errors.New("boom")))

		rec := testRequest(t, r, http.MethodGet, "/route")
		assert.Equal(t, rec.Code, http.StatusInternalServerError)
	})

	t.Run("group inherits error handlers", func(t *testing.T) {
		r := NewRouter()
		r.RegisterErrorHandler(func(w http.ResponseWriter, _ *http.Request, _ error) HandleState {
			w.WriteHeader(http.StatusTeapot)
			return HandleStop
		})
		r.GroupPrefix("/api", func(g *Router) {
			g.Get("/route", testErrorHandler(errors.New("boom")))
		})

		rec := testRequest(t, r, http.MethodGet, "/api/route")
		assert.Equal(t, rec.Code, http.StatusTeapot)
	})

	t.Run("handler receives the route error", func(t *testing.T) {
		want := errors.New("boom")

		r := NewRouter()
		r.RegisterErrorHandler(func(w http.ResponseWriter, _ *http.Request, err error) HandleState {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return HandleStop
		})
		r.Get("/route", testErrorHandler(want))

		rec := testRequest(t, r, http.MethodGet, "/route")
		assert.Equal(t, rec.Code, http.StatusBadRequest)
		assert.Equal(t, rec.Body.String(), want.Error()+"\n")
	})
}

func TestRouterInvalidRequestBody(t *testing.T) {
	r := NewRouter()
	r.Get("/route", testErrorHandler(ErrInvalidRequestBody))

	rec := testRequest(t, r, http.MethodGet, "/route")
	assert.Equal(t, rec.Code, http.StatusBadRequest)
	assert.Equal(t, rec.Body.String(), http.StatusText(http.StatusBadRequest)+"\n")
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "route not found", err: ErrRouteNotFound, want: http.StatusNotFound},
		{name: "wrapped route not found", err: fmt.Errorf("wrap: %w", ErrRouteNotFound), want: http.StatusNotFound},
		{name: "method not allowed", err: ErrMethodNotAllowed, want: http.StatusMethodNotAllowed},
		{name: "wrapped method not allowed", err: fmt.Errorf("wrap: %w", ErrMethodNotAllowed), want: http.StatusMethodNotAllowed},
		{name: "invalid request body", err: ErrInvalidRequestBody, want: http.StatusBadRequest},
		{name: "wrapped invalid request body", err: fmt.Errorf("wrap: %w", ErrInvalidRequestBody), want: http.StatusBadRequest},
		{name: "unhandled error", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			writeError(rec, tt.err)

			assert.Equal(t, rec.Code, tt.want)
			assert.Equal(t, rec.Body.String(), http.StatusText(tt.want)+"\n")
		})
	}
}
