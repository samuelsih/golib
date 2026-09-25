package oas

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/guregu/null/v6"
	"github.com/samuelsih/golib/assert"
	"github.com/samuelsih/golib/httpx"
)

func testHandler(body string) httpx.Handler {
	return func(w http.ResponseWriter, _ *http.Request) error {
		_, err := io.WriteString(w, body)
		return err
	}
}

func findParam(t *testing.T, params []RefT[Parameter], in ParameterIn, name string) *Parameter {
	t.Helper()

	for p := range slices.Values(params) {
		if p.Value == nil {
			continue
		}
		if p.Value.In == in && p.Value.Name == name {
			return p.Value
		}
	}

	t.Fatalf("parameter %s %q not found", in, name)
	return nil
}

type getUserParams struct {
	ID     string `path:"id" description:"User id"`
	Full   bool   `query:"full" default:"false" required:"false" description:"Include details"`
	APIKey string `header:"X-Api-Key" required:"true"`
	Limit  *int   `query:"limit" minimum:"1" maximum:"100" example:"20"`
}

type userModel struct {
	ID    string      `json:"id" description:"Unique id"`
	Name  string      `json:"name" minLength:"1" maxLength:"100" example:"Ada"`
	Email null.String `json:"email"`
	Age   *int        `json:"age,omitempty"`
	Tags  []string    `json:"tags,omitempty"`
}

type createUserModel struct {
	Name  string `json:"name" minLength:"1"`
	Email string `json:"email" format:"email" required:"true"`
}

func TestAPIServerOpenAPI(t *testing.T) {
	router := httpx.NewRouter(httpx.WithPathPrefix("/api/v1"))
	server := NewServer(router, ServerConfig{
		Title:   "Users API",
		Version: "2.0.0",
		Contact: null.ValueFrom(Contact{Name: null.StringFrom("API team")}),
		License: null.ValueFrom(License{Name: "MIT"}),
	})

	server.Get("/users/{id}", testHandler("user")).Spec(Spec{
		Summary: "Get user",
		Tags:    []string{"users"},
		Params:  SpecParams[getUserParams](),
		ExternalDocs: null.ValueFrom(ExternalDocumentation{
			URL: "https://example.com/users",
		}),
		Responses: []ResponseSpec{
			{Status: 200, Body: SpecBody[userModel]()},
			{Status: 404, Description: "Not found"},
		},
	})

	server.GroupPrefix("/admin", func(g *APIServer) {
		g.Post("/users", testHandler("created")).Spec(Spec{
			Body:      SpecBody[createUserModel](),
			Responses: []ResponseSpec{{Status: 201, Body: SpecBody[userModel]()}},
		})
	})

	doc, err := server.OpenAPI()
	assert.NoError(t, err)
	assert.Equal(t, doc.OpenAPI, "3.1.0")
	assert.Equal(t, doc.Info.Title, "Users API")
	assert.Equal(t, doc.Info.Version, "2.0.0")
	assert.Equal(t, doc.Info.Contact.V.Name.String, "API team")
	assert.Equal(t, doc.Info.License.V.Name, "MIT")

	getPath, ok := doc.Paths["/api/v1/users/{id}"]
	assert.True(t, ok)
	assert.True(t, getPath.Value.Get.Valid)

	op := getPath.Value.Get.V
	assert.Equal(t, op.Summary.String, "Get user")
	assert.Equal(t, op.Tags, []string{"users"})
	assert.Equal(t, op.ExternalDocs.V.URL, "https://example.com/users")

	id := findParam(t, op.Parameters, ParameterInPath, "id")
	assert.True(t, id.Required.Bool)
	assert.Equal(t, id.Description.String, "User id")

	full := findParam(t, op.Parameters, ParameterInQuery, "full")
	assert.False(t, full.Required.Bool)
	assert.Equal(t, full.Schema.Value.Default, false)
	assert.Equal(t, full.Description.String, "Include details")

	apiKey := findParam(t, op.Parameters, ParameterInHeader, "X-Api-Key")
	assert.True(t, apiKey.Required.Bool)

	limit := findParam(t, op.Parameters, ParameterInQuery, "limit")
	assert.False(t, limit.Required.Bool)
	assert.Equal(t, limit.Schema.Value.Minimum.Float64, float64(1))
	assert.Equal(t, limit.Schema.Value.Maximum.Float64, float64(100))
	assert.Equal(t, limit.Example, any(int64(20)))

	assert.Equal(t, op.Responses["404"].Value.Description, "Not found")

	ok200, ok := op.Responses["200"]
	assert.True(t, ok)
	assert.Equal(t, ok200.Value.Content["application/json"].Schema.Ref.Ref, "#/components/schemas/userModel")

	postPath, ok := doc.Paths["/api/v1/admin/users"]
	assert.True(t, ok)

	requestBody := postPath.Value.Post.V.RequestBody
	assert.True(t, requestBody.Value.Required.Bool)
	assert.Equal(t, requestBody.Value.Content["application/json"].Schema.Ref.Ref, "#/components/schemas/createUserModel")

	user := doc.Components.V.Schemas["userModel"].Value
	assert.Equal(t, user.Required, []string{"id", "name", "email"})
	assert.Equal(t, user.Properties["id"].Value.Description.String, "Unique id")
	assert.Equal(t, user.Properties["name"].Value.Examples[0], any("Ada"))
	assert.Equal(t, user.Properties["name"].Value.MinLength.Int64, int64(1))
	assert.Equal(t, user.Properties["name"].Value.MaxLength.Int64, int64(100))
	assert.Equal(t, user.Properties["email"].Value.Type, Strings{"string", "null"})
	assert.Equal(t, user.Properties["age"].Value.Type, Strings{"integer", "null"})
}

func TestAPIServerAutoPathParameterAndDefaults(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{Title: "Files"})

	server.Get("/files/{path...}", testHandler("file")).Spec(Spec{})
	server.Query("/search", testHandler("search")).Spec(Spec{})

	doc, err := server.OpenAPI()
	assert.NoError(t, err)

	item, ok := doc.Paths["/files/{path}"]
	assert.True(t, ok)

	op := item.Value.Get.V
	pathParam := findParam(t, op.Parameters, ParameterInPath, "path")
	assert.True(t, pathParam.Required.Bool)
	assert.Equal(t, pathParam.Schema.Value.Type, Strings{"string"})

	assert.Equal(t, op.Responses["default"].Value.Description, "Default response")

	_, ok = doc.Paths["/search"]
	assert.False(t, ok)
}

type nullableModel struct {
	Text  null.String        `json:"text"`
	Count null.Int           `json:"count"`
	Flag  null.Bool          `json:"flag"`
	Any   null.Value[string] `json:"any"`
	Ptr   *string            `json:"ptr"`
}

func TestNullableSchemas(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{Title: "Null"})

	server.Post("/things", testHandler("")).Spec(Spec{Body: SpecBody[nullableModel]()})

	doc, err := server.OpenAPI()
	assert.NoError(t, err)

	schema := doc.Components.V.Schemas["nullableModel"].Value
	assert.Equal(t, schema.Properties["text"].Value.Type, Strings{"string", "null"})
	assert.Equal(t, schema.Properties["count"].Value.Type, Strings{"integer", "null"})
	assert.Equal(t, schema.Properties["flag"].Value.Type, Strings{"boolean", "null"})
	assert.Equal(t, schema.Properties["any"].Value.Type, Strings{"string", "null"})
	assert.Equal(t, schema.Properties["ptr"].Value.Type, Strings{"string", "null"})
}

func TestAPIServerServeHTTP(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{Title: "Ping"})
	server.Get("/ping", testHandler("pong")).Spec(Spec{})

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ping", nil))

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.String(), "pong")
}

func TestNewServerInsideRouterGroup(t *testing.T) {
	router := httpx.NewRouter()

	router.GroupPrefix("/api", func(r *httpx.Router) {
		server := NewServer(r, ServerConfig{Title: "Grouped"})
		server.Get("/ping", testHandler("pong")).Spec(Spec{})

		doc := server.MustOpenAPI()
		_, ok := doc.Paths["/api/ping"]
		assert.True(t, ok)
	})
}

func TestOpenAPIErrors(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{})

	server.Get("/users/{id}", testHandler("")).Spec(Spec{
		Params: SpecParams[struct {
			Other string   `path:"other"`
			Bad   chan int `query:"bad"`
		}](),
	})

	_, err := server.OpenAPI()
	assert.NotNil(t, err)
}

func TestNestedAndEmbeddedSchemas(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{Title: "Articles"})

	server.Post("/articles", testHandler("")).Spec(Spec{
		Body: SpecBody[articleModel](),
		Responses: []ResponseSpec{
			{Status: 200, Body: SpecBody[listResponse]()},
		},
	})

	doc, err := server.OpenAPI()
	assert.NoError(t, err)

	article := doc.Components.V.Schemas["articleModel"].Value
	_, ok := article.Properties["secret"]
	assert.False(t, ok)
	_, ok = article.Properties[""]
	assert.False(t, ok)
	assert.False(t, slices.Contains(article.Required, ""))

	createdAt := article.Properties["createdAt"].Value
	assert.NotNil(t, createdAt)
	assert.Equal(t, createdAt.Type, Strings{"string"})
	assert.Equal(t, createdAt.Format.String, "date-time")

	owner := article.Properties["owner"].Value
	assert.Equal(t, owner.Description.String, "Article owner")
	assert.Equal(t, owner.AllOf[0].Ref.Ref, "#/components/schemas/userModel")

	list := doc.Components.V.Schemas["listResponse"].Value
	assert.Equal(t, list.Properties["items"].Value.Type, Strings{"array"})
	assert.Equal(t, list.Properties["items"].Value.Items.Ref.Ref, "#/components/schemas/userModel")
}

type auditFields struct {
	CreatedAt time.Time `json:"createdAt"`
}

type articleModel struct {
	auditFields
	Title  string    `json:"title"`
	Owner  userModel `json:"owner" description:"Article owner"`
	Secret string    `json:"-"`
}

type listResponse struct {
	Items []userModel `json:"items"`
}

func TestBodySpecOptions(t *testing.T) {
	body := SpecBody[userModel]()
	assert.True(t, body.required)
	assert.Equal(t, body.contentType, "application/json")

	body = body.Optional().WithContentType("application/xml").WithDescription("payload")
	assert.False(t, body.required)
	assert.Equal(t, body.contentType, "application/xml")
	assert.Equal(t, body.description, "payload")

	body = body.WithRequired(true).WithContentType("")
	assert.True(t, body.required)
	assert.Equal(t, body.contentType, "application/xml")

	assert.False(t, SpecBody[*userModel]().required)
}

func TestSpecBodyPointerSchema(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{Title: "Bodies"})

	server.Post("/users", testHandler("")).Spec(Spec{Body: SpecBody[*createUserModel]()})
	server.Get("/users/{id}", testHandler("")).Spec(Spec{
		Responses: []ResponseSpec{{Status: 200, Body: SpecBody[*userModel]()}},
	})

	doc, err := server.OpenAPI()
	assert.NoError(t, err)

	body := doc.Paths["/users"].Value.Post.V.RequestBody.Value
	assert.False(t, body.Required.Bool)
	assert.Equal(t, body.Content["application/json"].Schema.Ref.Ref, "#/components/schemas/createUserModel")

	response := doc.Paths["/users/{id}"].Value.Get.V.Responses["200"].Value
	assert.Equal(t, response.Content["application/json"].Schema.Ref.Ref, "#/components/schemas/userModel")
}

func TestAPIServerDocumentMarshal(t *testing.T) {
	router := httpx.NewRouter()
	server := NewServer(router, ServerConfig{Title: "Users"})
	server.Get("/users/{id}", testHandler("")).Spec(Spec{
		Params:    SpecParams[getUserParams](),
		Responses: []ResponseSpec{{Status: 200, Body: SpecBody[userModel]()}},
	})

	raw, err := json.Marshal(server.MustOpenAPI())
	assert.NoError(t, err)
	assert.True(t, len(raw) > 0)
}
