package server

import (
	"net/http"

	"github.com/swaggest/swgui/v5emb"

	apidocs "github.com/CreateFutureMWilkinson/cue/docs/api"
)

// docsAPIBasePath is the base path under which Swagger UI and the
// embedded API documentation files are served.
const docsAPIBasePath = "/docs/api/"

// registerDocsRoutes mounts the API documentation surface:
//
//   - GET /docs/api               -> Swagger UI (HTML)
//   - GET /docs/api/              -> Swagger UI (HTML, trailing slash)
//   - GET /docs/api/openapi.yaml  -> embedded OpenAPI specification
//   - GET /docs/api/websocket     -> embedded WebSocket reference (Markdown)
//
// The OpenAPI specification and WebSocket reference are embedded into the
// binary via the apidocs package so the docs surface works without any
// runtime file dependency.
func (s *Server) registerDocsRoutes() {
	swaggerUI := v5emb.New(
		"Cue API",
		docsAPIBasePath+apidocs.OpenAPISpecFile,
		docsAPIBasePath,
	)

	// Swagger UI HTML at both /docs/api and /docs/api/.
	s.mux.Handle("GET /docs/api", swaggerUI)
	s.mux.Handle("GET /docs/api/", swaggerUI)

	// Serve the embedded OpenAPI spec.
	s.mux.HandleFunc("GET "+docsAPIBasePath+apidocs.OpenAPISpecFile, func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, r, apidocs.OpenAPISpecFile, "application/yaml; charset=utf-8")
	})

	// Serve the embedded WebSocket reference (Markdown).
	s.mux.HandleFunc("GET /docs/api/websocket", func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, r, apidocs.WebSocketRefFile, "text/plain; charset=utf-8")
	})
}

// serveEmbeddedFile writes an embedded apidocs file with the given content type.
func serveEmbeddedFile(w http.ResponseWriter, _ *http.Request, name, contentType string) {
	data, err := apidocs.FS.ReadFile(name)
	if err != nil {
		http.Error(w, "documentation asset not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}
