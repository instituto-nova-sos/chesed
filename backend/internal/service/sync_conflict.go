package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// This file holds the S12.2 conflict-enrichment logic for the sync push path.
// On a conflict the server best-effort attaches a lean, non-PII projection of
// the existing server row (server_data / server_updated_at) so the offline
// client can offer field-level resolution. Every lookup failure falls back to
// the plain {status, message} conflict result, preserving Phase 1 back-compat.

// enrichPersonConflict augments a person conflict with a lean, non-PII
// projection of the existing server row. The row is resolvable only when the
// pushed payload carries the conflicting email; document/phone-only conflicts
// (and lookup failures) fall back to the plain conflict result.
func (s *SyncService) enrichPersonConflict(ctx context.Context, base domain.SyncPushResult, email *string) domain.SyncPushResult {
	if email == nil || *email == "" {
		return base
	}
	campusID := auth.CampusIDFromContext(ctx)
	existing, err := s.personRepo.FindByEmail(ctx, *email, campusID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			slog.WarnContext(ctx, "syncService: person conflict lookup failed", "error", err.Error())
		}
		return base
	}
	return withServerData(base, existing.ID, personConflictProjection(existing), existing.UpdatedAt)
}

// enrichTriageConflict augments a triage conflict with a lean projection of the
// existing server row, resolved by sync_id.
func (s *SyncService) enrichTriageConflict(ctx context.Context, rec domain.SyncPushRecord, campusID uuid.UUID, cause error) domain.SyncPushResult {
	base := conflictResult(rec.SyncID, nil, cause.Error())
	existing, err := s.triageRepo.FindBySyncID(ctx, rec.SyncID, campusID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			slog.WarnContext(ctx, "syncService: triage conflict lookup failed", "error", err.Error())
		}
		return base
	}
	return withServerData(base, existing.ID, triageConflictProjection(existing), existing.UpdatedAt)
}

// enrichAttendanceConflict augments an attendance conflict with a lean
// projection of the existing server row, resolved by sync_id.
func (s *SyncService) enrichAttendanceConflict(ctx context.Context, rec domain.SyncPushRecord, campusID uuid.UUID, cause error) domain.SyncPushResult {
	base := conflictResult(rec.SyncID, nil, cause.Error())
	existing, err := s.attendanceRepo.FindBySyncID(ctx, rec.SyncID, campusID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			slog.WarnContext(ctx, "syncService: attendance conflict lookup failed", "error", err.Error())
		}
		return base
	}
	return withServerData(base, existing.ID, attendanceConflictProjection(existing), existing.UpdatedAt)
}

// personConflictProjection returns only the non-PII-safe fields the sync client
// already holds (full_name, email, phone). It deliberately omits document
// number, birth date, gender, and every other sensitive attribute.
func personConflictProjection(p *domain.Person) map[string]any {
	m := map[string]any{"full_name": p.FullName}
	if p.Email != nil {
		m["email"] = *p.Email
	}
	if p.Phone != nil {
		m["phone"] = *p.Phone
	}
	return m
}

// triageConflictProjection returns only the fields the sync client already holds
// (main_complaint, triage_date). Clinical notes and other sensitive attributes
// are deliberately omitted.
func triageConflictProjection(t *domain.Triage) map[string]any {
	return map[string]any{
		"main_complaint": t.MainComplaint,
		"triage_date":    t.TriageDate,
	}
}

// attendanceConflictProjection returns only the fields the sync client already
// holds (status, attendance_date). Observations and recommendations may carry
// clinical detail and are deliberately omitted.
func attendanceConflictProjection(a *domain.Attendance) map[string]any {
	return map[string]any{
		"status":          a.Status,
		"attendance_date": a.AttendanceDate,
	}
}

// withServerData copies base and attaches the resolved server row's ID, lean
// projection, and updated_at (S12.2). ConflictFields is intentionally left unset
// — the client diffs the lean projection against its own local copy.
func withServerData(base domain.SyncPushResult, serverID uuid.UUID, data map[string]any, updatedAt time.Time) domain.SyncPushResult {
	id := serverID
	updated := updatedAt
	base.ServerID = &id
	base.ServerData = data
	base.ServerUpdatedAt = &updated
	return base
}
