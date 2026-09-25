package oas

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/guregu/null/v6"
	"github.com/samuelsih/golib/httpx"
	"github.com/samuelsih/golib/reflectx"
)

// ServerConfig holds the document-level metadata used to build the OpenAPI
// document. Version defaults to "1.0.0" when empty.
type ServerConfig struct {
	Title             string
	Summary           string
	Description       string
	TermsOfService    string
	Version           string
	Contact           null.Value[Contact]
	License           null.Value[License]
	Servers           []Server
	Tags              []Tag
	Security          []SecurityRequirement
	JSONSchemaDialect string
}

// APIServer wraps an *httpx.Router and records OpenAPI metadata for routes
// registered through it. It only builds documentation; the underlying router
// serves requests. Register routes before serving: registration and document
// building are not safe to run concurrently with each other.
type APIServer struct {
	router *httpx.Router
	config ServerConfig
	prefix string
	routes *routeSet
}

// routeSet is the ordered route list shared by a server and its groups.
type routeSet struct {
	items []*Route
}

// Route is a handle for a registered route. Attach the OpenAPI operation
// metadata with Spec.
type Route struct {
	method string
	path   string
	spec   Spec
}

// Spec sets the OpenAPI operation metadata for the route.
func (r *Route) Spec(spec Spec) *Route {
	r.spec = spec
	return r
}

// NewServer creates an APIServer on top of router. Generated paths include the
// router prefix and any prefix added with GroupPrefix.
func NewServer(router *httpx.Router, config ServerConfig) *APIServer {
	if router == nil {
		panic("oas: NewServer called with a nil router")
	}

	return &APIServer{
		router: router,
		config: config,
		prefix: router.Prefix(),
		routes: &routeSet{},
	}
}

// Router returns the underlying httpx router.
func (s *APIServer) Router() *httpx.Router {
	return s.router
}

// Use appends httpx middlewares to the underlying router.
func (s *APIServer) Use(middlewares ...httpx.Middleware) {
	s.router.Use(middlewares...)
}

// Group calls fn with a server whose routes share the parent prefix.
func (s *APIServer) Group(fn func(*APIServer)) {
	s.GroupPrefix("", fn)
}

// GroupPrefix calls fn with a server whose routes are registered under prefix.
func (s *APIServer) GroupPrefix(prefix string, fn func(*APIServer)) {
	if fn == nil {
		return
	}

	s.router.GroupPrefix(prefix, func(r *httpx.Router) {
		fn(&APIServer{
			router: r,
			config: s.config,
			prefix: path.Join(s.prefix, prefix),
			routes: s.routes,
		})
	})
}

// Get registers a GET route and returns it for documentation.
func (s *APIServer) Get(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle(http.MethodGet, route, handler, middlewares)
}

// Post registers a POST route and returns it for documentation.
func (s *APIServer) Post(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle(http.MethodPost, route, handler, middlewares)
}

// Put registers a PUT route and returns it for documentation.
func (s *APIServer) Put(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle(http.MethodPut, route, handler, middlewares)
}

// Delete registers a DELETE route and returns it for documentation.
func (s *APIServer) Delete(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle(http.MethodDelete, route, handler, middlewares)
}

// Head registers a HEAD route and returns it for documentation.
func (s *APIServer) Head(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle(http.MethodHead, route, handler, middlewares)
}

// Options registers an OPTIONS route and returns it for documentation.
func (s *APIServer) Options(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle(http.MethodOptions, route, handler, middlewares)
}

// Trace registers a TRACE route and returns it for documentation.
func (s *APIServer) Trace(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle(http.MethodTrace, route, handler, middlewares)
}

// Query registers a QUERY route and returns it for documentation. QUERY has
// no representation in OpenAPI 3.1, so the route is registered on the router
// but omitted from the document.
func (s *APIServer) Query(route string, handler httpx.Handler, middlewares ...httpx.Middleware) *Route {
	return s.handle("QUERY", route, handler, middlewares)
}

func (s *APIServer) handle(method, route string, handler httpx.Handler, middlewares []httpx.Middleware) *Route {
	switch method {
	case http.MethodGet:
		s.router.Get(route, handler, middlewares...)
	case http.MethodPost:
		s.router.Post(route, handler, middlewares...)
	case http.MethodPut:
		s.router.Put(route, handler, middlewares...)
	case http.MethodDelete:
		s.router.Delete(route, handler, middlewares...)
	case http.MethodHead:
		s.router.Head(route, handler, middlewares...)
	case http.MethodOptions:
		s.router.Options(route, handler, middlewares...)
	case http.MethodTrace:
		s.router.Trace(route, handler, middlewares...)
	case "QUERY":
		s.router.Query(route, handler, middlewares...)
	}

	full := path.Join(s.prefix, route)
	if full == "" {
		full = "/"
	}

	r := &Route{method: method, path: full}
	s.routes.items = append(s.routes.items, r)

	return r
}

// ServeHTTP delegates to the underlying httpx router.
func (s *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Spec describes a single route for documentation purposes. The zero Spec is
// valid and produces an operation with a default response.
type Spec struct {
	Summary      string
	Description  string
	Tags         []string
	OperationID  string
	Deprecated   bool
	Params       ParamsSpec
	Body         BodySpec
	Responses    []ResponseSpec
	Security     []SecurityRequirement
	Servers      []Server
	ExternalDocs null.Value[ExternalDocumentation]
}

// ParamsSpec describes the parameters of an operation. Use SpecParams to build
// one from a struct type.
type ParamsSpec struct {
	types []reflect.Type
}

// SpecParams derives operation parameters from T's path, query, header and
// cookie tags. Path params are required; others unless a pointer or required tag.
func SpecParams[T any]() ParamsSpec {
	return ParamsSpec{types: []reflect.Type{reflect.TypeFor[T]()}}
}

// And appends another struct type to the parameter spec.
func (p ParamsSpec) And[T any]() ParamsSpec {
	p.types = append(slices.Clone(p.types), reflect.TypeFor[T]())
	return p
}

// BodySpec describes a request or response body. Use SpecBody to build one
// from a struct type.
type BodySpec struct {
	typeOf      reflect.Type
	required    bool
	contentType string
	description string
}

// SpecBody derives the request or response body schema from T. The default
// content type is application/json. A pointer T marks the body optional.
func SpecBody[T any]() BodySpec {
	t := reflect.TypeFor[T]()

	return BodySpec{
		typeOf:      t,
		required:    t.Kind() != reflect.Pointer,
		contentType: "application/json",
	}
}

// Optional marks the body as not required.
func (b BodySpec) Optional() BodySpec {
	b.required = false
	return b
}

// WithRequired sets whether the body is required.
func (b BodySpec) WithRequired(required bool) BodySpec {
	b.required = required
	return b
}

// WithContentType overrides the default application/json content type.
func (b BodySpec) WithContentType(contentType string) BodySpec {
	if contentType != "" {
		b.contentType = contentType
	}

	return b
}

// WithDescription sets the request body description.
func (b BodySpec) WithDescription(description string) BodySpec {
	b.description = description
	return b
}

// ResponseSpec describes a single response of an operation.
type ResponseSpec struct {
	// Status is the HTTP status code. Zero means the "default" response.
	Status      int
	Description string
	Body        BodySpec
}

// OpenAPI builds the OpenAPI 3.1 document from the routes registered on the
// server. The returned document is non-nil; err reports invalid input.
func (s *APIServer) OpenAPI() (*OpenAPI, error) {
	cfg := s.config

	var errs []error
	if cfg.Title == "" {
		errs = append(errs, errors.New("oas: ServerConfig.Title is required"))
	}

	version := cfg.Version
	if version == "" {
		version = "1.0.0"
	}

	info := Info{Title: cfg.Title, Version: version}
	if cfg.Summary != "" {
		info.Summary = null.StringFrom(cfg.Summary)
	}
	if cfg.Description != "" {
		info.Description = null.StringFrom(cfg.Description)
	}
	if cfg.TermsOfService != "" {
		info.TermsOfService = null.StringFrom(cfg.TermsOfService)
	}
	if cfg.Contact.Valid {
		info.Contact = cfg.Contact
	}
	if cfg.License.Valid {
		info.License = cfg.License
	}

	doc := &OpenAPI{OpenAPI: "3.1.0", Info: info}
	if cfg.JSONSchemaDialect != "" {
		doc.JSONSchemaDialect = null.StringFrom(cfg.JSONSchemaDialect)
	}
	if len(cfg.Servers) > 0 {
		doc.Servers = cfg.Servers
	}
	if len(cfg.Tags) > 0 {
		doc.Tags = cfg.Tags
	}
	if len(cfg.Security) > 0 {
		doc.Security = cfg.Security
	}

	b := newSchemaBuilder()
	paths := Paths{}
	seen := map[string]struct{}{}

	for route := range slices.Values(s.routes.items) {
		openPath, pathParams, err := parsePath(route.path)
		if err != nil {
			b.addErr(err)
			continue
		}

		key := route.method + " " + openPath
		if _, ok := seen[key]; ok {
			b.addErr(fmt.Errorf("oas: duplicate route %s %s", route.method, openPath))
			continue
		}
		seen[key] = struct{}{}

		item, ok := paths[openPath]
		if !ok {
			item = RefT[PathItem]{Value: &PathItem{}}
		}

		if !setOperation(item.Value, route.method, buildOperation(b, route.spec, pathParams)) {
			continue
		}
		paths[openPath] = item
	}

	if len(paths) > 0 {
		doc.Paths = paths
	}
	if len(b.comps) > 0 {
		doc.Components = null.ValueFrom(Components{Schemas: b.comps})
	}

	errs = append(errs, b.errs...)

	return doc, errors.Join(errs...)
}

// MustOpenAPI is like OpenAPI but panics when the document cannot be built.
func (s *APIServer) MustOpenAPI() *OpenAPI {
	doc, err := s.OpenAPI()
	if err != nil {
		panic(err)
	}

	return doc
}

func setOperation(item *PathItem, method string, op Operation) bool {
	v := null.ValueFrom(op)

	switch method {
	case http.MethodGet:
		item.Get = v
	case http.MethodPut:
		item.Put = v
	case http.MethodPost:
		item.Post = v
	case http.MethodDelete:
		item.Delete = v
	case http.MethodOptions:
		item.Options = v
	case http.MethodHead:
		item.Head = v
	case http.MethodPatch:
		item.Patch = v
	case http.MethodTrace:
		item.Trace = v
	default:
		// QUERY and other custom methods have no OpenAPI representation.
		return false
	}

	return true
}

func buildOperation(b *schemaBuilder, spec Spec, pathParams []string) Operation {
	op := Operation{
		Tags:     spec.Tags,
		Security: spec.Security,
		Servers:  spec.Servers,
	}
	if spec.Summary != "" {
		op.Summary = null.StringFrom(spec.Summary)
	}
	if spec.Description != "" {
		op.Description = null.StringFrom(spec.Description)
	}
	if spec.OperationID != "" {
		op.OperationID = null.StringFrom(spec.OperationID)
	}
	if spec.Deprecated {
		op.Deprecated = null.BoolFrom(true)
	}
	if spec.ExternalDocs.Valid {
		op.ExternalDocs = spec.ExternalDocs
	}

	if params := buildParameters(b, spec.Params, pathParams); len(params) > 0 {
		op.Parameters = params
	}

	if spec.Body.typeOf != nil {
		body := &RequestBody{
			Required: null.BoolFrom(spec.Body.required),
			Content: map[string]MediaType{
				spec.Body.contentType: {Schema: b.build(reflectx.Indirect(spec.Body.typeOf))},
			},
		}
		if spec.Body.description != "" {
			body.Description = null.StringFrom(spec.Body.description)
		}

		op.RequestBody = RefT[RequestBody]{Value: body}
	}

	op.Responses = buildResponses(b, spec.Responses)

	return op
}

func buildResponses(b *schemaBuilder, specs []ResponseSpec) Responses {
	if len(specs) == 0 {
		return Responses{
			"default": RefT[Response]{Value: &Response{Description: "Default response"}},
		}
	}

	responses := Responses{}
	for rs := range slices.Values(specs) {
		key := "default"
		description := rs.Description

		if rs.Status != 0 {
			if rs.Status < 100 || rs.Status > 599 {
				b.addErr(fmt.Errorf("oas: invalid response status %d", rs.Status))
				continue
			}

			key = strconv.Itoa(rs.Status)
			if description == "" {
				description = http.StatusText(rs.Status)
			}
		}
		if description == "" {
			description = "Response"
		}

		response := Response{Description: description}
		if rs.Body.typeOf != nil {
			response.Content = map[string]MediaType{
				rs.Body.contentType: {Schema: b.build(reflectx.Indirect(rs.Body.typeOf))},
			}
		}

		responses[key] = RefT[Response]{Value: &response}
	}

	return responses
}

var parameterLocations = [...]struct {
	in  ParameterIn
	tag string
}{
	{ParameterInPath, "path"},
	{ParameterInQuery, "query"},
	{ParameterInHeader, "header"},
	{ParameterInCookie, "cookie"},
}

func buildParameters(b *schemaBuilder, params ParamsSpec, pathParams []string) []RefT[Parameter] {
	var out []RefT[Parameter]

	seen := map[string]struct{}{}
	add := func(p Parameter) {
		key := string(p.In) + ":" + p.Name
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, RefT[Parameter]{Value: &p})
	}

	declaredPath := map[string]struct{}{}

	for t := range slices.Values(params.types) {
		t = reflectx.Indirect(t)
		if t.Kind() != reflect.Struct {
			b.addErr(fmt.Errorf("oas: SpecParams requires a struct type, got %s", t))
			continue
		}

		reflectx.WalkFields(t, shouldFlattenParams, func(sf reflect.StructField) {
			in, name, ok := parameterLocation(b, sf)
			if !ok {
				return
			}

			p := Parameter{
				Name:     name,
				In:       in,
				Required: null.BoolFrom(parameterRequired(sf, in)),
				Schema:   applyFieldTags(b.build(sf.Type), sf, true),
			}
			if desc, ok := reflectx.GetTagValueAs[string](sf, "description"); ok && desc != "" {
				p.Description = null.StringFrom(desc)
			}

			if v, ok := reflectx.GetTagValueAs[bool](sf, "deprecated"); ok {
				p.Deprecated = null.BoolFrom(v)
			}
			if v, ok := reflectx.GetTagValueAs[bool](sf, "allowEmptyValue"); ok {
				p.AllowEmptyValue = null.BoolFrom(v)
			}
			if v, ok := reflectx.GetTagValueAs[bool](sf, "allowReserved"); ok {
				p.AllowReserved = null.BoolFrom(v)
			}
			if v, ok := reflectx.GetTagValueAs[bool](sf, "explode"); ok {
				p.Explode = null.BoolFrom(v)
			}
			if v, ok := reflectx.GetTagValueAs[string](sf, "style"); ok && v != "" {
				p.Style = null.ValueFrom(ParameterStyle(v))
			}
			if v, ok := typedTagValue(sf, "example", schemaValueType(p.Schema.Value)); ok {
				p.Example = v
			}

			if in == ParameterInPath {
				declaredPath[name] = struct{}{}
			}

			add(p)
		})
	}

	for name := range slices.Values(pathParams) {
		if _, ok := declaredPath[name]; ok {
			continue
		}

		add(Parameter{
			Name:     name,
			In:       ParameterInPath,
			Required: null.BoolFrom(true),
			Schema:   RefT[Schema]{Value: &Schema{Type: Strings{"string"}}},
		})
	}

	for name := range declaredPath {
		if !slices.Contains(pathParams, name) {
			b.addErr(fmt.Errorf("oas: path parameter %q is not present in the route", name))
		}
	}

	return out
}

func parameterLocation(b *schemaBuilder, sf reflect.StructField) (ParameterIn, string, bool) {
	var (
		in    ParameterIn
		name  string
		found int
	)

	for loc := range slices.Values(parameterLocations[:]) {
		value, ok := sf.Tag.Lookup(loc.tag)
		if !ok {
			continue
		}

		found++
		if found == 1 {
			in = loc.in
			name = value
			if name == "" {
				name = sf.Name
			}
		}
	}

	if found == 0 {
		return "", "", false
	}
	if found > 1 {
		b.addErr(fmt.Errorf("oas: field %s has multiple parameter location tags", sf.Name))
	}

	return in, name, true
}

func parameterRequired(sf reflect.StructField, in ParameterIn) bool {
	if v, ok := reflectx.GetTagValueAs[bool](sf, "required"); ok {
		return v
	}
	if in == ParameterInPath {
		return true
	}

	return sf.Type.Kind() != reflect.Pointer
}

func shouldFlattenParams(sf reflect.StructField) bool {
	return !hasLocationTag(sf.Tag)
}

func hasLocationTag(tag reflect.StructTag) bool {
	for loc := range slices.Values(parameterLocations[:]) {
		if _, ok := tag.Lookup(loc.tag); ok {
			return true
		}
	}

	return false
}

// parsePath converts a Go 1.22 ServeMux pattern into an OpenAPI path and
// returns the names of its path parameters.
func parsePath(route string) (openPath string, params []string, err error) {
	var out strings.Builder

	for i := 0; i < len(route); {
		if route[i] != '{' {
			out.WriteByte(route[i])
			i++
			continue
		}

		end := strings.IndexByte(route[i:], '}')
		if end < 0 {
			return "", nil, fmt.Errorf("oas: unmatched '{' in route %q", route)
		}

		segment := route[i+1 : i+end]
		i += end + 1

		if segment == "$" {
			continue
		}

		name := strings.TrimSuffix(segment, "...")
		if name == "" || strings.ContainsAny(name, "{}") {
			return "", nil, fmt.Errorf("oas: invalid path parameter %q in route %q", segment, route)
		}

		params = append(params, name)
		out.WriteByte('{')
		out.WriteString(name)
		out.WriteByte('}')
	}

	openPath = out.String()
	if openPath == "" {
		openPath = "/"
	}

	return openPath, params, nil
}
