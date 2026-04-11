// Package apierrors defines sentinel errors shared across service and API layers.
package apierrors

import "errors"

// ErrNotFound is returned by service methods when a requested resource does not exist.
// Handlers map this to HTTP 404 via errors.Is.
var ErrNotFound = errors.New("not found")

// ErrInvalidInput is returned by service methods when user input is invalid (e.g. invalid mcp tool name).
var ErrInvalidInput = errors.New("invalid user input")
