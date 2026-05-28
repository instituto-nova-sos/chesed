package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAgreementChecker struct{ mock.Mock }

func (m *mockAgreementChecker) HasAcceptedAgreement(ctx context.Context, personID uuid.UUID) (bool, error) {
	args := m.Called(ctx, personID)
	return args.Bool(0), args.Error(1)
}

type mockRoleFinder struct{ mock.Mock }

func (m *mockRoleFinder) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.PersonRole, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PersonRole), args.Error(1)
}

func TestRequireAgreement(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("passes through when no person_id in claims", func(t *testing.T) {
		checker := new(mockAgreementChecker)
		finder := new(mockRoleFinder)
		mw := RequireAgreement(checker, finder)

		claims := auth.AuthClaims{Subject: uuid.New().String(), Roles: []string{"VOLUNTEER"}}
		ctx := auth.NewContext(context.Background(), claims)

		req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		mw(okHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("passes through when person has no VOLUNTEER role", func(t *testing.T) {
		checker := new(mockAgreementChecker)
		finder := new(mockRoleFinder)
		mw := RequireAgreement(checker, finder)

		personID := uuid.New()
		claims := auth.AuthClaims{
			Subject:  uuid.New().String(),
			PersonID: personID,
			Roles:    []string{"ADMIN"},
		}
		ctx := auth.NewContext(context.Background(), claims)

		finder.On("FindByPersonID", mock.Anything, personID).
			Return([]domain.PersonRole{{RoleType: "ADMIN", IsActive: true}}, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		mw(okHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("blocks volunteer without accepted agreement", func(t *testing.T) {
		checker := new(mockAgreementChecker)
		finder := new(mockRoleFinder)
		mw := RequireAgreement(checker, finder)

		personID := uuid.New()
		claims := auth.AuthClaims{
			Subject:  uuid.New().String(),
			PersonID: personID,
			Roles:    []string{"VOLUNTEER"},
		}
		ctx := auth.NewContext(context.Background(), claims)

		finder.On("FindByPersonID", mock.Anything, personID).
			Return([]domain.PersonRole{{RoleType: "VOLUNTEER", IsActive: true}}, nil)
		checker.On("HasAcceptedAgreement", mock.Anything, personID).Return(false, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		mw(okHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)

		var body map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Equal(t, "agreement_required", body["error"])
	})

	t.Run("allows volunteer with accepted agreement", func(t *testing.T) {
		checker := new(mockAgreementChecker)
		finder := new(mockRoleFinder)
		mw := RequireAgreement(checker, finder)

		personID := uuid.New()
		claims := auth.AuthClaims{
			Subject:  uuid.New().String(),
			PersonID: personID,
			Roles:    []string{"VOLUNTEER"},
		}
		ctx := auth.NewContext(context.Background(), claims)

		finder.On("FindByPersonID", mock.Anything, personID).
			Return([]domain.PersonRole{{RoleType: "VOLUNTEER", IsActive: true}}, nil)
		checker.On("HasAcceptedAgreement", mock.Anything, personID).Return(true, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		mw(okHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("ignores inactive volunteer role", func(t *testing.T) {
		checker := new(mockAgreementChecker)
		finder := new(mockRoleFinder)
		mw := RequireAgreement(checker, finder)

		personID := uuid.New()
		claims := auth.AuthClaims{
			Subject:  uuid.New().String(),
			PersonID: personID,
			Roles:    []string{"VOLUNTEER"},
		}
		ctx := auth.NewContext(context.Background(), claims)

		finder.On("FindByPersonID", mock.Anything, personID).
			Return([]domain.PersonRole{{RoleType: "VOLUNTEER", IsActive: false}}, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		mw(okHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		// HasAcceptedAgreement should NOT be called since role is inactive
		checker.AssertNotCalled(t, "HasAcceptedAgreement")
	})
}
