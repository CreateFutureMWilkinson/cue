// Package apidocs embeds the Cue REST OpenAPI specification and the
// WebSocket reference Markdown so they can be served by the HTTP server
// without depending on files on disk.
//
// The embed lives in this package (at the repository root, alongside
// openapi.yaml and websocket.md) because Go's embed directive cannot
// traverse parent directories from a file inside internal/.
package apidocs

import "embed"

// FS holds the embedded API documentation files.
//
//go:embed openapi.yaml websocket.md
var FS embed.FS

// OpenAPISpecFile is the filename of the embedded OpenAPI specification.
const OpenAPISpecFile = "openapi.yaml"

// WebSocketRefFile is the filename of the embedded WebSocket reference.
const WebSocketRefFile = "websocket.md"
