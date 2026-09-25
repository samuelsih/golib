package pbd

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"net/http"
)

// ContentType is the media type for RFC 9457 problem details responses.
const ContentType = "application/problem+json"

// Option configures a Problem.
type Option func(*Problem)

// Problem is an RFC 9457 problem details object.
type Problem struct { //nolint:errname // "problem details" is the RFC 9457 name for this object.
	Type     string         `json:"type,omitempty"`
	Status   int            `json:"status"`
	Title    string         `json:"title,omitempty"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Code     string         `json:"code,omitempty"`
	Errors   map[string]any `json:"errors,omitempty"`

	// Custom holds RFC 9457 extension members promoted into the object.
	Custom map[string]any `json:"-"`
}

// New creates a default Problem for the given HTTP status code.
func New(statusCode int, opts ...Option) *Problem {
	p := &Problem{
		Type:   "about:blank",
		Status: statusCode,
		Title:  http.StatusText(statusCode),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Error implements error.
func (p *Problem) Error() string {
	title := p.Title
	if title == "" {
		title = http.StatusText(p.Status)
	}

	if p.Detail == "" {
		return fmt.Sprintf("%d %s", p.Status, title)
	}

	return fmt.Sprintf("%d %s: %s", p.Status, title, p.Detail)
}

// Write writes the problem to w as application/problem+json using its status code.
func (p *Problem) Write(w http.ResponseWriter) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(p.Status)

	_, err = w.Write(body)

	return err
}

// MarshalJSONTo implements json.MarshalerTo for encoding/json/v2.
func (p Problem) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plain Problem

	raw, err := json.Marshal(plain(p))
	if err != nil {
		return err
	}

	if len(p.Custom) == 0 {
		return enc.WriteValue(raw)
	}

	members := make(map[string]jsontext.Value, len(p.Custom))
	if err := json.Unmarshal(raw, &members); err != nil {
		return err
	}

	for key, value := range p.Custom {
		if _, exists := members[key]; exists {
			return fmt.Errorf("pbd: custom member %q conflicts with an existing member", key)
		}

		if members[key], err = json.Marshal(value); err != nil {
			return err
		}
	}

	merged, err := json.Marshal(members)
	if err != nil {
		return err
	}

	return enc.WriteValue(merged)
}

// MarshalJSON implements json.Marshaler for encoding/json v1.
func (p Problem) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	if err := p.MarshalJSONTo(jsontext.NewEncoder(&buf)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WithType sets the problem type URI reference.
func WithType(uri string) Option {
	return func(p *Problem) {
		p.Type = uri
	}
}

// WithTitle sets a short, human-readable summary of the problem type.
func WithTitle(title string) Option {
	return func(p *Problem) {
		p.Title = title
	}
}

// WithDetail sets a human-readable explanation specific to this occurrence.
func WithDetail(detail string) Option {
	return func(p *Problem) {
		p.Detail = detail
	}
}

// WithInstance sets a URI reference identifying this specific occurrence.
func WithInstance(uri string) Option {
	return func(p *Problem) {
		p.Instance = uri
	}
}

// WithCode sets an application-specific error code extension member.
func WithCode(code string) Option {
	return func(p *Problem) {
		p.Code = code
	}
}

// WithErrors sets the errors extension member. Values may be nested maps or
// any JSON-marshalable value.
func WithErrors(errs map[string]any) Option {
	return func(p *Problem) {
		p.Errors = errs
	}
}

// WithCustom merges RFC 9457 extension members promoted into the top level.
// Keys conflicting with members set by other options return a marshal error.
func WithCustom(custom map[string]any) Option {
	return func(p *Problem) {
		if p.Custom == nil {
			p.Custom = make(map[string]any, len(custom))
		}

		maps.Copy(p.Custom, custom)
	}
}
