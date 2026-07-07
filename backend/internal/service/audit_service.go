package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// AuditRepository defines the interface for audit log persistence.
type AuditRepository interface {
	Create(ctx context.Context, entry domain.AuditLog) error
}

// AuditParams holds parameters for creating an audit log entry.
type AuditParams struct {
	ActionType  string
	EntityType  string
	EntityID    *uuid.UUID
	Module      string
	Description string
	OldValues   any
	NewValues   any
	IPAddress   string
	UserAgent   string
	Success     bool
}

// AuditService handles audit log creation.
type AuditService struct {
	repo AuditRepository
}

// NewAuditService creates a new AuditService.
func NewAuditService(repo AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// LogAction creates an audit log entry.
// User and campus are extracted from the request context.
func (s *AuditService) LogAction(ctx context.Context, params AuditParams) error {
	entry, err := buildAuditEntry(ctx, params)
	if err != nil {
		return fmt.Errorf("auditService.LogAction: %w", err)
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return fmt.Errorf("auditService.LogAction: %w", err)
	}

	return nil
}

// LogRequired writes an audit entry whose persistence is mandatory: a failure is
// returned to the caller so the per-request campus transaction rolls back the
// mutation it accompanies. It is reserved for LGPD/compliance-critical actions
// (erasure and export of personal data) where a mutation without its audit
// record is a compliance liability, not a tolerable availability trade-off. See
// docs/adr/0001-audit-logging-durability.md. Ordinary mutations must keep using
// the best-effort per-service audit helpers instead.
func (s *AuditService) LogRequired(ctx context.Context, params AuditParams) error {
	if err := s.LogAction(ctx, params); err != nil {
		return fmt.Errorf("auditService.LogRequired: %w", err)
	}
	return nil
}

func buildAuditEntry(ctx context.Context, params AuditParams) (domain.AuditLog, error) {
	claims := auth.ClaimsFromContext(ctx)

	entry := domain.AuditLog{
		ID:         uuid.New(),
		ActionType: params.ActionType,
		EntityType: params.EntityType,
		EntityID:   params.EntityID,
		Success:    params.Success,
		Timestamp:  time.Now(),
	}

	if claims.Subject != "" {
		if userID, err := uuid.Parse(claims.Subject); err == nil {
			entry.UserID = &userID
		}
	}
	if claims.CampusID != uuid.Nil {
		entry.CampusID = &claims.CampusID
	}

	setOptionalString(&entry.Module, params.Module)
	setOptionalString(&entry.Description, params.Description)
	setOptionalString(&entry.IPAddress, params.IPAddress)
	setOptionalString(&entry.UserAgent, params.UserAgent)

	if params.OldValues != nil {
		data, err := json.Marshal(params.OldValues)
		if err != nil {
			return domain.AuditLog{}, fmt.Errorf("marshal old_values: %w", err)
		}
		entry.OldValues = data
	}
	if params.NewValues != nil {
		data, err := json.Marshal(params.NewValues)
		if err != nil {
			return domain.AuditLog{}, fmt.Errorf("marshal new_values: %w", err)
		}
		entry.NewValues = data
	}

	return entry, nil
}

func setOptionalString(dst **string, val string) {
	if val != "" {
		*dst = &val
	}
}
