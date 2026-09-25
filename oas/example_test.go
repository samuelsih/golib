package oas

import (
	"net/http"

	"github.com/samuelsih/golib/httpx"
)

type exampleUser struct {
	ID   string `json:"id" example:"u_1"`
	Name string `json:"name" description:"Display name" example:"Ada"`
}

type exampleCreateUser struct {
	Name string `json:"name" minLength:"1" description:"Display name" example:"Ada"`
}

type exampleListParams struct {
	Page int `query:"page" default:"1" minimum:"1" description:"Page number" example:"2"`
}

// ExampleNewServer registers documented routes on an httpx router and builds
// the OpenAPI document. Handlers keep the plain httpx.Handler signature.
func ExampleNewServer() {
	router := httpx.NewRouter(httpx.WithPathPrefix("/api/v1"))
	server := NewServer(router, ServerConfig{
		Title:   "Users API",
		Version: "1.0.0",
	})

	server.Get("/users", func(http.ResponseWriter, *http.Request) error {
		return nil
	}).Spec(Spec{
		Summary: "List users",
		Tags:    []string{"users"},
		Params:  SpecParams[exampleListParams](),
		Responses: []ResponseSpec{
			{Status: 200, Body: SpecBody[[]exampleUser]()},
		},
	})

	server.Post("/users", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusCreated)
		return nil
	}).Spec(Spec{
		Summary: "Create user",
		Tags:    []string{"users"},
		Body:    SpecBody[exampleCreateUser](),
		Responses: []ResponseSpec{
			{Status: 201, Body: SpecBody[exampleUser]()},
		},
	})

	doc := server.MustOpenAPI()
	_ = doc
}
