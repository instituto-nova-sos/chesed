package domain

import "errors"

// Sentinel errors for domain-level error handling.
var (
	ErrNotFound     = errors.New("not found")
	ErrDuplicate    = errors.New("duplicate")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
)
