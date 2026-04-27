// Command api-lint validates an OpenAPI 3.1 specification file.
//
// Usage: api-lint <path-to-openapi.yaml>
//
// Exits 0 on success, non-zero on validation failure.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

// Validate loads the OpenAPI 3.1 YAML file at path and validates it using
// github.com/getkin/kin-openapi/openapi3. External $ref resolution is
// disabled so that validation is hermetic and operates only on the local
// document.
func Validate(ctx context.Context, path string) error {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}

	if err := doc.Validate(ctx); err != nil {
		return fmt.Errorf("validate %s: %w", path, err)
	}

	return nil
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
