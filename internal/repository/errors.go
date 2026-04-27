package repository

import "errors"

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrNotImplemented is returned by stubs that have not yet been implemented.
var ErrNotImplemented = errors.New("not implemented")

// ErrDuplicate is returned when a unique-constraint violation prevents
// creating or renaming an entity (e.g. a category whose key already exists).
var ErrDuplicate = errors.New("duplicate")

// ErrInvalidRoutingRule is returned when a routing rule fails validation.
var ErrInvalidRoutingRule = errors.New("invalid routing rule")

// ErrValidation marks errors caused by invalid user-supplied input
// (e.g. malformed category names or colour codes). Handlers wrap this
// sentinel to map service errors to HTTP 400 without string matching.
var ErrValidation = errors.New("validation")
