package oas

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samuelsih/golib/assert"
	"github.com/samuelsih/golib/httpx"
)

func TestEnableDocUIDefaultRoute(t *testing.T) {
	router := httpx.NewRouter(httpx.WithPathPrefix("/api/v1"))
	server := NewServer(router, ServerConfig{Title: "Users API"})
	server.Get("/users", testHandler("users")).Spec(Spec{Summary: "List users"})

	server.EnableDocUI("")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/docs", nil))

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.True(t, strings.Contains(rec.Body.String(), "/api/v1/docs.json"))
	assert.True(t, strings.Contains(rec.Body.String(), "<title>Users API</title>"))

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/docs.json", nil))

	assert.Equal(t, rec.Code, http.StatusOK)

	var doc OpenAPI
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	assert.Equal(t, doc.Info.Title, "Users API")

	_, ok := doc.Paths["/api/v1/users"]
	assert.True(t, ok)

	_, ok = doc.Paths["/api/v1/docs"]
	assert.False(t, ok)
}

func TestEnableDocUICustomRoute(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{Title: "Ping"})

	server.EnableDocUI("/reference/")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/reference", nil))

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.True(t, strings.Contains(rec.Body.String(), `apiDescriptionUrl="/reference.json"`))

	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/reference.json", nil))

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.True(t, strings.Contains(rec.Body.String(), `"openapi":"3.1.0"`))
}
