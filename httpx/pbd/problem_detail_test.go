package pbd

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/samuelsih/golib/assert"
)

func TestNewDefaults(t *testing.T) {
	p := New(http.StatusNotFound)

	assert.Equal(t, p.Status, http.StatusNotFound)
	assert.Equal(t, p.Title, "Not Found")
	assert.Equal(t, p.Type, "about:blank")
	assert.Nil(t, p.Errors)
}

func TestNewOptions(t *testing.T) {
	p := New(http.StatusUnprocessableEntity,
		WithType("https://example.com/problems/validation"),
		WithTitle("Validation failed"),
		WithDetail("email is invalid"),
		WithInstance("/requests/123"),
		WithCode("invalid_request"),
		WithErrors(map[string]any{
			"email": []string{"required", "invalid"},
			"profile": map[string]any{
				"age": "must be positive",
			},
		}),
	)

	assert.Equal(t, p.Type, "https://example.com/problems/validation")
	assert.Equal(t, p.Title, "Validation failed")
	assert.Equal(t, p.Detail, "email is invalid")
	assert.Equal(t, p.Instance, "/requests/123")
	assert.Equal(t, p.Code, "invalid_request")
	assert.True(t, reflect.DeepEqual(p.Errors["email"], []string{"required", "invalid"}))
}

func TestProblemMarshalsRFC9457(t *testing.T) {
	p := New(http.StatusUnprocessableEntity,
		WithType("https://example.com/problems/validation"),
		WithDetail("email is invalid"),
		WithCode("invalid_request"),
		WithErrors(map[string]any{
			"profile": map[string]any{"age": "must be positive"},
		}),
	)

	got := marshalToMap(t, p)

	want := map[string]any{
		"type":   "https://example.com/problems/validation",
		"status": float64(http.StatusUnprocessableEntity),
		"title":  "Unprocessable Entity",
		"detail": "email is invalid",
		"code":   "invalid_request",
		"errors": map[string]any{
			"profile": map[string]any{"age": "must be positive"},
		},
	}

	assert.Equal(t, reflect.DeepEqual(got, want), true)
}

func TestProblemOmitsEmptyMembers(t *testing.T) {
	got := marshalToMap(t, New(http.StatusBadRequest))

	want := map[string]any{
		"type":   "about:blank",
		"status": float64(http.StatusBadRequest),
		"title":  "Bad Request",
	}

	assert.Equal(t, reflect.DeepEqual(got, want), true)
}

func TestProblemError(t *testing.T) {
	assert.Equal(t, New(http.StatusNotFound, WithDetail("user not found")).Error(), "404 Not Found: user not found")
	assert.Equal(t, New(http.StatusBadRequest).Error(), "400 Bad Request")
	assert.Equal(t, New(http.StatusBadRequest, WithTitle("Oops")).Error(), "400 Oops")
	assert.Equal(t, New(http.StatusBadRequest, WithTitle("")).Error(), "400 Bad Request")
}

func TestProblemWrite(t *testing.T) {
	rec := httptest.NewRecorder()

	err := New(http.StatusTeapot, WithDetail("no coffee")).Write(rec)
	assert.NoError(t, err)

	assert.Equal(t, rec.Code, http.StatusTeapot)
	assert.Equal(t, rec.Header().Get("Content-Type"), ContentType)

	got := map[string]any{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	want := map[string]any{
		"type":   "about:blank",
		"status": float64(http.StatusTeapot),
		"title":  "I'm a teapot",
		"detail": "no coffee",
	}

	assert.Equal(t, reflect.DeepEqual(got, want), true)
}

func TestWithCustomInlinesExtensions(t *testing.T) {
	p := New(http.StatusTooManyRequests,
		WithCode("rate_limited"),
		WithCustom(map[string]any{
			"retryAfter": 30,
			"meta":       map[string]any{"requestId": "abc"},
		}),
	)

	got := marshalToMap(t, p)

	want := map[string]any{
		"type":       "about:blank",
		"status":     float64(http.StatusTooManyRequests),
		"title":      "Too Many Requests",
		"code":       "rate_limited",
		"retryAfter": float64(30),
		"meta":       map[string]any{"requestId": "abc"},
	}

	assert.Equal(t, reflect.DeepEqual(got, want), true)
}

func TestWithCustomMerges(t *testing.T) {
	p := New(http.StatusBadRequest,
		WithDetail("original"),
		WithCustom(map[string]any{"a": 1}),
		WithCustom(map[string]any{"b": 2}),
	)

	got := marshalToMap(t, p)

	want := map[string]any{
		"type":   "about:blank",
		"status": float64(http.StatusBadRequest),
		"title":  "Bad Request",
		"detail": "original",
		"a":      float64(1),
		"b":      float64(2),
	}

	assert.Equal(t, reflect.DeepEqual(got, want), true)
}

func TestWithCustomFillsUnsetMembers(t *testing.T) {
	got := marshalToMap(t, New(http.StatusBadRequest, WithCustom(map[string]any{"detail": "custom detail"})))

	want := map[string]any{
		"type":   "about:blank",
		"status": float64(http.StatusBadRequest),
		"title":  "Bad Request",
		"detail": "custom detail",
	}

	assert.Equal(t, reflect.DeepEqual(got, want), true)
}

func TestWithCustomConflictErrors(t *testing.T) {
	rec := httptest.NewRecorder()

	p := New(http.StatusBadRequest, WithDetail("original"), WithCustom(map[string]any{"detail": "clobbered"}))
	assert.NotNil(t, p.Write(rec))
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.Len(), 0)
}

func TestWithCustomMarshalError(t *testing.T) {
	rec := httptest.NewRecorder()

	err := New(http.StatusBadRequest, WithCustom(map[string]any{"bad": make(chan int)})).Write(rec)
	assert.NotNil(t, err)
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.Len(), 0)
}

func TestMarshalJSONSupportsV1(t *testing.T) {
	p := New(http.StatusTeapot, WithCustom(map[string]any{"traceId": "abc"}))

	raw, err := jsonv1.Marshal(p)
	assert.NoError(t, err)

	got := map[string]any{}
	assert.NoError(t, json.Unmarshal(raw, &got))

	want := map[string]any{
		"type":    "about:blank",
		"status":  float64(http.StatusTeapot),
		"title":   "I'm a teapot",
		"traceId": "abc",
	}

	assert.Equal(t, reflect.DeepEqual(got, want), true)
}

func marshalToMap(t *testing.T, p *Problem) map[string]any {
	t.Helper()

	raw, err := json.Marshal(p)
	assert.NoError(t, err)

	got := map[string]any{}
	assert.NoError(t, json.Unmarshal(raw, &got))

	return got
}
