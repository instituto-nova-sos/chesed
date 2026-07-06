package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// retentionWindow is the operational-data retention period (docs/13): personal
// data older than this with no recent activity is anonymized.
const retentionWindow = 5 * 365 * 24 * time.Hour

// RetentionRepository lists records whose retention window has lapsed.
type RetentionRepository interface {
	ListExpiredPersonIDs(ctx context.Context, campusID uuid.UUID, olderThan time.Time) ([]uuid.UUID, error)
}

// RetentionSummary reports the outcome of a retention sweep.
type RetentionSummary struct {
	Scanned    int `json:"scanned"`
	Anonymized int `json:"anonymized"`
}

// RetentionService enforces the LGPD data-retention policy by anonymizing
// operational personal data past its retention window. It reuses the same
// PersonAnonymizer as consent revocation (S11.3), so anonymization semantics
// are identical.
type RetentionService struct {
	repo       RetentionRepository
	anonymizer PersonAnonymizer
	auditSvc   *AuditService
}

// NewRetentionService creates a new RetentionService.
func NewRetentionService(repo RetentionRepository, anonymizer PersonAnonymizer, auditSvc *AuditService) *RetentionService {
	return &RetentionService{repo: repo, anonymizer: anonymizer, auditSvc: auditSvc}
}

// Run anonymizes every campus-scoped person whose last activity is older than
// the retention window, auditing each. Already-anonymized records are excluded
// by the repository query, making repeated runs idempotent.
func (s *RetentionService) Run(ctx context.Context) (RetentionSummary, error) {
	campusID := auth.CampusIDFromContext(ctx)
	if campusID == uuid.Nil {
		return RetentionSummary{}, fmt.Errorf("retentionService.Run: %w", domain.ErrForbidden)
	}

	cutoff := time.Now().UTC().Add(-retentionWindow)
	ids, err := s.repo.ListExpiredPersonIDs(ctx, campusID, cutoff)
	if err != nil {
		return RetentionSummary{}, fmt.Errorf("retentionService.Run: %w", err)
	}

	summary := RetentionSummary{Scanned: len(ids)}
	for _, id := range ids {
		if err := s.anonymizer.Anonymize(ctx, id, campusID); err != nil {
			return summary, fmt.Errorf("retentionService.Run: anonymize %s: %w", id, err)
		}
		summary.Anonymized++
		s.audit(ctx, id)
	}
	return summary, nil
}

// audit records a single retention anonymization without recording any PII.
func (s *RetentionService) audit(ctx context.Context, personID uuid.UUID) {
	params := AuditParams{
		ActionType:  "UPDATE",
		EntityType:  "person",
		EntityID:    &personID,
		Module:      "retention",
		Description: "person anonymized (LGPD retention policy)",
		NewValues:   map[string]any{"anonymized_at": time.Now().UTC()},
		Success:     true,
	}
	if auditErr := s.auditSvc.LogAction(ctx, params); auditErr != nil {
		slog.ErrorContext(ctx, "retentionService: audit failed", "error", auditErr.Error())
	}
}
