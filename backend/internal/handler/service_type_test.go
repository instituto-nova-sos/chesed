package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockServiceTypeRepo implements service.ServiceTypeRepository.
type mockServiceTypeRepo struct {
	mock.Mock
}

func (m *mockServiceTypeRepo) ListActive(ctx context.Context) ([]domain.ServiceType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.ServiceType), args.Error(1)
}

func TestServiceTypeHandler_List(t *testing.T) {
	now := time.Now()

	t.Run("returns 200 with data", func(t *testing.T) {
		repo := new(mockServiceTypeRepo)
		svc := service.NewServiceTypeService(repo)
		h := NewServiceTypeHandler(svc)

		items := []domain.ServiceType{
			{ID: uuid.New(), Name: "Consulta Medica", Category: "MEDICAL", IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), Name: "Fisioterapia", Category: "PHYSIOTHERAPY", IsActive: true, CreatedAt: now, UpdatedAt: now},
		}
		repo.On("ListActive", mock.Anything).Return(items, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/service-types", nil)
		rec := httptest.NewRecorder()
		h.List(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var body map[string][]domain.ServiceType
		err := json.Unmarshal(rec.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Len(t, body["data"], 2)
		assert.Equal(t, "Consulta Medica", body["data"][0].Name)
		repo.AssertExpectations(t)
	})

	t.Run("returns 200 with empty list", func(t *testing.T) {
		repo := new(mockServiceTypeRepo)
		svc := service.NewServiceTypeService(repo)
		h := NewServiceTypeHandler(svc)

		repo.On("ListActive", mock.Anything).Return([]domain.ServiceType{}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/service-types", nil)
		rec := httptest.NewRecorder()
		h.List(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string][]domain.ServiceType
		err := json.Unmarshal(rec.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Len(t, body["data"], 0)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		repo := new(mockServiceTypeRepo)
		svc := service.NewServiceTypeService(repo)
		h := NewServiceTypeHandler(svc)

		repo.On("ListActive", mock.Anything).Return(nil, errors.New("db down"))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/service-types", nil)
		rec := httptest.NewRecorder()
		h.List(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		var body map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, "internal_error", body["error"])
		assert.Equal(t, "failed to list service types", body["message"])
	})
}
