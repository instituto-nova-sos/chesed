package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicCampusSource(t *testing.T) {
	campusID := uuid.New()

	tests := []struct {
		name   string
		ctx    context.Context
		wantID uuid.UUID
		wantOK bool
	}{
		{
			name:   "campus present",
			ctx:    WithPublicCampus(context.Background(), campusID),
			wantID: campusID,
			wantOK: true,
		},
		{
			name:   "campus absent",
			ctx:    context.Background(),
			wantID: uuid.Nil,
			wantOK: false,
		},
		{
			name:   "nil campus stored",
			ctx:    WithPublicCampus(context.Background(), uuid.Nil),
			wantID: uuid.Nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(tt.ctx)
			id, ok := publicCampusSource(req)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

// stubCampusRepo satisfies PublicCampusLookup for validator tests.
type stubCampusRepo struct {
	campus *domain.Campus
	err    error
}

func (s stubCampusRepo) FindByID(_ context.Context, _ uuid.UUID) (*domain.Campus, error) {
	return s.campus, s.err
}

func activeCampus(id uuid.UUID) *domain.Campus {
	return &domain.Campus{ID: id, Name: "Test", IsActive: true}
}

func TestPublicCampusValidator_GET(t *testing.T) {
	campusID := uuid.New()

	tests := []struct {
		name       string
		query      string
		repo       stubCampusRepo
		wantStatus int
		wantCode   string
	}{
		{
			name:       "valid active campus",
			query:      "campus_id=" + campusID.String(),
			repo:       stubCampusRepo{campus: activeCampus(campusID)},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing campus_id",
			query:      "",
			repo:       stubCampusRepo{campus: activeCampus(campusID)},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "malformed campus_id",
			query:      "campus_id=not-a-uuid",
			repo:       stubCampusRepo{campus: activeCampus(campusID)},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "unknown campus",
			query:      "campus_id=" + campusID.String(),
			repo:       stubCampusRepo{err: domain.ErrNotFound},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "inactive campus",
			query:      "campus_id=" + campusID.String(),
			repo:       stubCampusRepo{campus: &domain.Campus{ID: campusID, IsActive: false}},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seenCampus uuid.UUID
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, ok := publicCampusSource(r)
				require.True(t, ok, "campus must be in context by the time next runs")
				seenCampus = id
				w.WriteHeader(http.StatusOK)
			})

			handler := PublicCampusValidator(tt.repo)(next)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/public/campaigns?"+tt.query, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, campusID, seenCampus)
				return
			}
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.wantCode, body["error"])
		})
	}
}

func TestPublicCampusValidator_POST_RestoresBody(t *testing.T) {
	campusID := uuid.New()
	repo := stubCampusRepo{campus: activeCampus(campusID)}

	payload := `{"full_name":"Ada Lovelace","campus_id":"` + campusID.String() + `"}`

	var seenBody string
	var seenCampus uuid.UUID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := publicCampusSource(r)
		require.True(t, ok)
		seenCampus = id
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		seenBody = string(b)
		w.WriteHeader(http.StatusCreated)
	})

	handler := PublicCampusValidator(repo)(next)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/volunteers", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, campusID, seenCampus)
	assert.JSONEq(t, payload, seenBody, "handler must still be able to read the full body")
}

func TestPublicCampusValidator_POST_Errors(t *testing.T) {
	campusID := uuid.New()

	tests := []struct {
		name       string
		body       string
		repo       stubCampusRepo
		wantStatus int
		wantCode   string
	}{
		{
			name:       "malformed json",
			body:       `{not json`,
			repo:       stubCampusRepo{campus: activeCampus(campusID)},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "missing campus_id",
			body:       `{"full_name":"X"}`,
			repo:       stubCampusRepo{campus: activeCampus(campusID)},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "unknown campus",
			body:       `{"campus_id":"` + campusID.String() + `"}`,
			repo:       stubCampusRepo{err: domain.ErrNotFound},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			})
			handler := PublicCampusValidator(tt.repo)(next)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/public/volunteers", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.wantCode, body["error"])
		})
	}
}
