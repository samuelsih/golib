package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"slices"
)

var (
	ErrRouteNotFound      = errors.New("route not found")
	ErrInvalidRequestBody = errors.New("invalid request body")
	ErrMethodNotAllowed   = errors.New("http method not allowed")
)

type (
	HandleState  int
	Handler      func(http.ResponseWriter, *http.Request) error
	ErrorHandler func(http.ResponseWriter, *http.Request, error) HandleState
	Middleware   func(Handler) Handler
	RouterOption func(*Router)
)

type Endpoint struct {
	Method     string
	PatternURL string
}

const (
	// HandleContinue tells the route error handler to continue on all registered error handler.
	HandleContinue HandleState = iota

	// HandleStop tells the route error handler to stop loop.
	HandleStop
)

// WithPathPrefix option add prefix routes in url.
func WithPathPrefix(prefix string) RouterOption {
	return func(r *Router) {
		r.prefix = prefix
	}
}

type Router struct {
	mux    *http.ServeMux
	chain  []Middleware
	onErrs []ErrorHandler
	state  *routerState
	prefix string
}

// routerState is shared by every Router derived from the same mux.
type routerState struct {
	endpoints []Endpoint
	paths     map[string]struct{}
}

// NewRouter create a new router with given RouterOptions.
func NewRouter(opts ...RouterOption) *Router {
	r := &Router{
		mux:   http.NewServeMux(),
		state: &routerState{paths: make(map[string]struct{})},
	}

	for _, o := range opts {
		o(r)
	}

	r.mux.Handle("/", r.wrap(r.notFound, nil))

	return r
}

// Prefix returns the router path prefix set by WithPathPrefix or GroupPrefix.
func (r *Router) Prefix() string {
	return r.prefix
}

// Use appends a middleware handler to the Router middleware stack. The chain
// is resolved per request, so it applies to routes registered before Use too.
func (r *Router) Use(middlewares ...Middleware) {
	r.chain = append(r.chain, middlewares...)
}

// RegisterErrorHandler appends an ErrorHandler to list of error handlers.
func (r *Router) RegisterErrorHandler(fn ErrorHandler) {
	if fn == nil {
		return
	}
	r.onErrs = append(r.onErrs, fn)
}

// Walk calls fn for every endpoint registered on the router.
func (r *Router) Walk(fn func(Endpoint)) {
	if fn == nil {
		return
	}

	for endpoint := range slices.Values(r.state.endpoints) {
		fn(endpoint)
	}
}

// Group creates a new router with a copy of middleware stack.
func (r *Router) Group(fn func(r *Router)) {
	r.GroupPrefix("", fn)
}

// GroupPrefix creates a new router with a copy of middleware stack + url prefix.
func (r *Router) GroupPrefix(prefix string, fn func(*Router)) {
	fn(&Router{
		mux:    r.mux,
		chain:  slices.Clone(r.chain),
		onErrs: slices.Clone(r.onErrs),
		state:  r.state,
		prefix: path.Join(r.prefix, prefix),
	})
}

// Get adds the route that matches an GET http method to
// execute the `fn` httpx.Handler.
func (r *Router) Get(route string, fn Handler, mx ...Middleware) {
	r.handle(http.MethodGet, route, fn, mx)
}

// Post adds the route that matches an POST http method to
// execute the `fn` httpx.Handler.
func (r *Router) Post(route string, fn Handler, mx ...Middleware) {
	r.handle(http.MethodPost, route, fn, mx)
}

// Put adds the route that matches an PUT http method to
// execute the `fn` httpx.Handler.
func (r *Router) Put(route string, fn Handler, mx ...Middleware) {
	r.handle(http.MethodPut, route, fn, mx)
}

// Delete adds the route that matches an DELETE http method to
// execute the `fn` httpx.Handler.
func (r *Router) Delete(route string, fn Handler, mx ...Middleware) {
	r.handle(http.MethodDelete, route, fn, mx)
}

// Head adds the route that matches an HEAD http method to
// execute the `fn` httpx.Handler.
func (r *Router) Head(route string, fn Handler, mx ...Middleware) {
	r.handle(http.MethodHead, route, fn, mx)
}

// Options adds the route that matches an OPTIONS http method to
// execute the `fn` httpx.Handler.
func (r *Router) Options(route string, fn Handler, mx ...Middleware) {
	r.handle(http.MethodOptions, route, fn, mx)
}

// Trace adds the route that matches an TRACE http method to
// execute the `fn` httpx.Handler.
func (r *Router) Trace(route string, fn Handler, mx ...Middleware) {
	r.handle(http.MethodTrace, route, fn, mx)
}

// Query adds the route that matches an QUERY http method to
// execute the `fn` httpx.Handler.
func (r *Router) Query(route string, fn Handler, mx ...Middleware) {
	r.handle("QUERY", route, fn, mx)
}

// Post adds the route that matches an POST http method to
// execute the `fn` httpx.Handler.
func (r *Router) handle(method, route string, fn Handler, mx []Middleware) {
	p := path.Join(r.prefix, route)

	pattern := p
	if p == "/" {
		p = "/{$}"
	}

	r.mux.Handle(method+" "+p, r.wrap(fn, mx))
	r.state.endpoints = append(r.state.endpoints, Endpoint{Method: method, PatternURL: pattern})

	if _, ok := r.state.paths[p]; ok {
		return
	}
	r.state.paths[p] = struct{}{}
	r.mux.Handle(p, r.wrap(r.methodNotAllowed, nil))
}

func (r *Router) notFound(_ http.ResponseWriter, _ *http.Request) error {
	return ErrRouteNotFound
}

func (r *Router) methodNotAllowed(_ http.ResponseWriter, _ *http.Request) error {
	return ErrMethodNotAllowed
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) wrap(fn Handler, mx []Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_ = r.buildHandler(fn, mx)(w, req)
	})
}

// buildHandler resolves the middleware chain at request time.
func (r *Router) buildHandler(fn Handler, mx []Middleware) Handler {
	chain := append(slices.Clone(r.chain), mx...)

	slices.Reverse(chain)

	fn = r.handleError(fn)
	for _, m := range chain {
		fn = r.handleError(m(fn))
	}

	return fn
}

// handledError marks an error whose response handleError already wrote, so
// outer middlewares observe it without writing a second response.
type handledError struct{ err error }

func (e *handledError) Error() string { return e.err.Error() }

func (e *handledError) Unwrap() error { return e.err }

// handleError runs the registered error handlers for any error fn returns,
// writing with the same writer fn received.
func (r *Router) handleError(fn Handler) Handler {
	return func(w http.ResponseWriter, req *http.Request) error {
		err := fn(w, req)
		if err == nil {
			return nil
		}

		if _, handled := errors.AsType[*handledError](err); handled {
			return err
		}

		for customErrHandler := range slices.Values(r.onErrs) {
			if customErrHandler(w, req, err) == HandleStop {
				return &handledError{err: err}
			}
		}

		writeError(w, err)

		return &handledError{err: err}
	}
}

func writeError(w http.ResponseWriter, err error) {
	_, bodyTooLarge := errors.AsType[*http.MaxBytesError](err)

	var status int
	switch {
	case errors.Is(err, ErrRouteNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrMethodNotAllowed):
		status = http.StatusMethodNotAllowed
	case errors.Is(err, ErrInvalidRequestBody):
		status = http.StatusBadRequest
		slog.Debug("json unmarshal error", slog.String("error", err.Error()))
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, ErrTimeoutExceeded):
		status = http.StatusGatewayTimeout
	case errors.Is(err, context.Canceled):
		status = 499
	case bodyTooLarge:
		status = http.StatusRequestEntityTooLarge
	default:
		status = http.StatusInternalServerError
		slog.Error("unhandled error", slog.String("error", err.Error()))
	}

	http.Error(w, http.StatusText(status), status)
}
