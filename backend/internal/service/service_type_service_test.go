package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockServiceTypeRepository is a testify mock for ServiceTypeRepository.
type MockServiceTypeRepository struct {
	mock.Mock
}

func (m *MockServiceTypeRepository) ListActive(ctx context.Context) ([]domain.ServiceType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.ServiceType), args.Error(1)
}

func TestServiceTypeService_ListActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		repoData  []domain.ServiceType
		repoErr   error
		wantLen   int
		wantErr   bool
		errSubstr string
	}{
		{
			name: "returns list of service types",
			repoData: []domain.ServiceType{
				{ID: uuid.New(), Name: "Consulta Medica", Category: "MEDICAL", IsActive: true, CreatedAt: now, UpdatedAt: now},
				{ID: uuid.New(), Name: "Fisioterapia", Category: "PHYSIOTHERAPY", IsActive: true, CreatedAt: now, UpdatedAt: now},
			},
			wantLen: 2,
		},
		{
			name:     "returns empty list",
			repoData: []domain.ServiceType{},
			wantLen:  0,
		},
		{
			name:      "repo error propagated",
			repoErr:   errors.New("db connection refused"),
			wantErr:   true,
			errSubstr: "db connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockServiceTypeRepository)
			svc := NewServiceTypeService(repo)

			if tt.repoErr != nil {
				repo.On("ListActive", mock.Anything).Return(nil, tt.repoErr)
			} else {
				repo.On("ListActive", mock.Anything).Return(tt.repoData, nil)
			}

			got, err := svc.ListActive(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
			}
			repo.AssertExpectations(t)
		})
	}
}
