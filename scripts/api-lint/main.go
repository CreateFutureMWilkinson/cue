// Command api-lint validates an OpenAPI 3.1 specification file.
//
// Usage: api-lint <path-to-openapi.yaml>
//
// Exits 0 on success, non-zero on validation failure.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// ErrNotImplemented is returned by the stub Validate implementation until
// the GREEN phase wires in the real kin-openapi/openapi3 validator.
var ErrNotImplemented = errors.New("not implemented")

// Validate loads the OpenAPI 3.1 YAML file at path and validates it.
//
// In GREEN, this will parse the document via github.com/getkin/kin-openapi/openapi3
// and run (*openapi3.T).Validate(ctx). For now it is a noop stub.
func Validate(ctx context.Context, path string) error {
	_ = ctx
	_ = path
	return ErrNotImplemented
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: api-lint <path-to-openapi.yaml>")
		os.Exit(2)
	}
	if err := Validate(context.Background(), os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "api-lint: %v\n", err)
		os.Exit(1)
	}
}
