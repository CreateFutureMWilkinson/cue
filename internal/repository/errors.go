package repository

import "errors"

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrNotImplemented is returned by stubs that have not yet been implemented.
var ErrNotImplemented = errors.New("not implemented")

// ErrInvalidRoutingRule is returned when a routing rule fails validation.
var ErrInvalidRoutingRule = errors.New("invalid routing rule")
