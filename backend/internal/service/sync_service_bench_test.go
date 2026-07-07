package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// BenchmarkPushBatch measures the sync push hot path (RNF-08, S12.5) against a
// representative 50-record batch. It exercises the real SyncService.PushSkipping
// Validation entrypoint with in-memory testify mocks, so it runs WITHOUT a live
// database. Every record is a fresh person that resolves to "not yet synced" and
// is then created, so the benchmark covers the per-record decode → validate →
// build → create → audit path that dominates a real batch push.
func BenchmarkPushBatch(b *testing.B) {
	const batchSize = 50

	campusID := uuid.New()
	ctx := auth.NewContext(context.Background(), auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "bench@example.com",
		Roles:    []string{"VOLUNTEER"},
		CampusID: campusID,
	})

	svc := newBenchSyncService()

	records := make([]domain.SyncPushRecord, 0, batchSize)
	for range batchSize {
		records = append(records, domain.SyncPushRecord{
			EntityType: domain.SyncEntityPerson,
			SyncID:     uuid.New(),
			Data: map[string]any{
				"full_name":     "Maria",
				"document_type": "CPF",
				"nationality":   "BRA",
			},
		})
	}

	b.ReportAllocs()
	for range b.N {
		if _, err := svc.PushSkippingValidation(ctx, records); err != nil {
			b.Fatalf("PushSkippingValidation: %v", err)
		}
	}
}

// newBenchSyncService wires a SyncService whose repositories are permissive
// mocks: every sync_id looks new (ErrNotFound), every create succeeds, and every
// audit write is a no-op. mock.Anything matchers keep the stubs valid across all
// b.N iterations without per-iteration setup cost polluting the measurement.
func newBenchSyncService() *SyncService {
	pRepo := new(MockSyncPersonRepo)
	tRepo := new(MockSyncTriageRepo)
	aRepo := new(MockSyncAttendanceRepo)
	cRepo := new(MockCampaignRepository)
	auditRepo := new(MockAuditRepository)

	pRepo.On("FindBySyncID", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, domain.ErrNotFound)
	pRepo.On("CreateWithSync", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.Person{ID: uuid.New()}, nil)
	auditRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	auditSvc := NewAuditService(auditRepo)
	return NewSyncService(pRepo, tRepo, aRepo, cRepo, auditSvc)
}
