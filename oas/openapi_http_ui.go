package oas

import (
	"encoding/json/v2"
	"html/template"
	"net/http"
	"path"
	"strings"
)

const defaultDocUIRoute = "/docs"

const docUIHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{.Title}}</title>
    <script src="https://unpkg.com/@stoplight/elements@9/web-components.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements@9/styles.min.css">
    <style>
      html,
      body {
        height: 100%;
        margin: 0;
      }

      elements-api {
        display: block;
        height: 100vh;
      }
    </style>
  </head>
  <body>
    <elements-api
      apiDescriptionUrl="{{.SpecURL}}"
      router="hash"
      layout="sidebar"
    ></elements-api>
  </body>
</html>
`

var docUITemplate = template.Must(template.New("doc_ui").Parse(docUIHTML))

type docUIPage struct {
	Title   string
	SpecURL string
}

// EnableDocUI serves a Stoplight Elements UI at route and its OpenAPI JSON at route+".json".
// An empty route defaults to "/docs"; neither route appears in the document.
func (s *APIServer) EnableDocUI(route string) {
	route = strings.TrimSuffix(route, "/")
	if route == "" {
		route = defaultDocUIRoute
	}

	specRoute := route + ".json"
	specURL := path.Join(s.prefix, specRoute)

	title := s.config.Title
	if title == "" {
		title = "API Documentation"
	}

	s.router.Get(route, func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		return docUITemplate.Execute(w, docUIPage{Title: title, SpecURL: specURL})
	})

	s.router.Get(specRoute, func(w http.ResponseWriter, _ *http.Request) error {
		doc, err := s.OpenAPI()
		if err != nil {
			return err
		}

		raw, err := json.Marshal(doc)
		if err != nil {
			return err
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, err = w.Write(raw)

		return err
	})
}
