package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// maxAuditPerPage caps the audit viewer page size to protect the database from
// unbounded scans over a table that grows without limit.
const maxAuditPerPage = 100

// AuditListRepository is the read side of the audit log used by the viewer.
// It is separate from the write-only AuditRepository interface so the append-only
// contract of audit_log stays explicit.
type AuditListRepository interface {
	List(ctx context.Context, filter domain.AuditLogFilter) (*domain.AuditLogListResult, error)
}

// AuditReadService serves the compliance audit log viewer. Campus scoping is
// enforced here (audit_log is not under RLS) by injecting the caller's campus
// into every query.
type AuditReadService struct {
	repo AuditListRepository
}

// NewAuditReadService creates a new AuditReadService.
func NewAuditReadService(repo AuditListRepository) *AuditReadService {
	return &AuditReadService{repo: repo}
}

// List returns audit entries for the caller's campus, applying pagination
// defaults and clamping the page size.
func (s *AuditReadService) List(ctx context.Context, filter domain.AuditLogFilter) (*domain.AuditLogListResult, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return nil, fmt.Errorf("auditReadService.List: %w", domain.ErrForbidden)
	}
	filter.CampusID = campusID

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 50
	}
	if filter.PerPage > maxAuditPerPage {
		filter.PerPage = maxAuditPerPage
	}

	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("auditReadService.List: %w", err)
	}
	return result, nil
}
