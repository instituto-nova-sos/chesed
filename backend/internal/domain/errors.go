package domain

import "errors"

// Sentinel errors for domain-level error handling.
var (
	ErrNotFound          = errors.New("not found")
	ErrDuplicate         = errors.New("duplicate")
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrDuplicatePhone    = errors.New("duplicate phone")
	ErrForbidden         = errors.New("forbidden")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidCPF        = errors.New("invalid CPF")
	ErrAgreementRequired = errors.New("volunteer agreement required")
	ErrAgreementExists   = errors.New("agreement already exists for this role")
)
