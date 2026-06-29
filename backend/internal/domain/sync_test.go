package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

func TestSyncPushRequestValidate(t *testing.T) {
	t.Parallel()

	validRecord := domain.SyncPushRecord{
		EntityType: domain.SyncEntityPerson,
		SyncID:     uuid.New(),
		Data:       map[string]any{"full_name": "Maria"},
	}

	tests := []struct {
		name    string
		records []domain.SyncPushRecord
		wantErr error
	}{
		{
			name:    "empty batch is valid no-op",
			records: []domain.SyncPushRecord{},
			wantErr: nil,
		},
		{
			name:    "batch at limit is valid",
			records: makeRecords(domain.MaxSyncBatchSize, validRecord),
			wantErr: nil,
		},
		{
			name:    "batch over limit is rejected",
			records: makeRecords(domain.MaxSyncBatchSize+1, validRecord),
			wantErr: domain.ErrBatchTooLarge,
		},
		{
			name: "unknown entity type is rejected",
			records: []domain.SyncPushRecord{{
				EntityType: "campaign",
				SyncID:     uuid.New(),
				Data:       map[string]any{"x": 1},
			}},
			wantErr: domain.ErrInvalidEntityType,
		},
		{
			name: "nil sync_id is rejected",
			records: []domain.SyncPushRecord{{
				EntityType: domain.SyncEntityPerson,
				SyncID:     uuid.Nil,
				Data:       map[string]any{"x": 1},
			}},
			wantErr: domain.ErrMissingSyncID,
		},
		{
			name: "empty data is rejected",
			records: []domain.SyncPushRecord{{
				EntityType: domain.SyncEntityPerson,
				SyncID:     uuid.New(),
				Data:       map[string]any{},
			}},
			wantErr: domain.ErrMissingData,
		},
		{
			name: "nil data is rejected",
			records: []domain.SyncPushRecord{{
				EntityType: domain.SyncEntityPerson,
				SyncID:     uuid.New(),
				Data:       nil,
			}},
			wantErr: domain.ErrMissingData,
		},
		{
			name: "triage and attendance entity types are valid",
			records: []domain.SyncPushRecord{
				{EntityType: domain.SyncEntityTriage, SyncID: uuid.New(), Data: map[string]any{"x": 1}},
				{EntityType: domain.SyncEntityAttendance, SyncID: uuid.New(), Data: map[string]any{"x": 1}},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := domain.SyncPushRequest{Records: tt.records}
			err := req.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
		})
	}
}

func TestSyncResultStatusConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "created", domain.SyncStatusCreated)
	assert.Equal(t, "conflict", domain.SyncStatusConflict)
	assert.Equal(t, "error", domain.SyncStatusError)
}

func TestSyncEntityTypeConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "person", domain.SyncEntityPerson)
	assert.Equal(t, "triage", domain.SyncEntityTriage)
	assert.Equal(t, "attendance", domain.SyncEntityAttendance)
}

func TestParseSyncEntityTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr error
	}{
		{
			name:  "single entity",
			input: "person",
			want:  []string{domain.SyncEntityPerson},
		},
		{
			name:  "multiple entities",
			input: "person,triage,attendance",
			want:  []string{domain.SyncEntityPerson, domain.SyncEntityTriage, domain.SyncEntityAttendance},
		},
		{
			name:  "tolerates whitespace",
			input: " person , triage ",
			want:  []string{domain.SyncEntityPerson, domain.SyncEntityTriage},
		},
		{
			name:    "empty string returns error",
			input:   "",
			wantErr: domain.ErrInvalidEntityType,
		},
		{
			name:    "unknown entity returns error",
			input:   "person,campaign",
			wantErr: domain.ErrInvalidEntityType,
		},
		{
			name:    "duplicate entries return error",
			input:   "person,person",
			wantErr: domain.ErrInvalidEntityType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseSyncEntityTypes(tt.input)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func makeRecords(n int, template domain.SyncPushRecord) []domain.SyncPushRecord {
	out := make([]domain.SyncPushRecord, n)
	for i := range out {
		r := template
		r.SyncID = uuid.New()
		out[i] = r
	}
	return out
}
