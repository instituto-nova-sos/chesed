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

	tests := []struct {
		name string
		// noPerson: claims carry no person_id (skips role lookup entirely).
		noPerson bool
		// roles returned by the finder; nil when noPerson is true.
		roles []domain.PersonRole
		// agreementAccepted: stubbed HasAcceptedAgreement result. nil = not stubbed.
		agreementAccepted *bool
		wantCode          int
		wantBlockedBody   bool
		assertNoChecker   bool
	}{
		{name: "passes through when no person_id", noPerson: true, wantCode: http.StatusOK},
		{name: "passes through when no VOLUNTEER role", roles: []domain.PersonRole{{RoleType: "ADMIN", IsActive: true}}, wantCode: http.StatusOK},
		{name: "blocks volunteer without accepted agreement", roles: []domain.PersonRole{{RoleType: "VOLUNTEER", IsActive: true}}, agreementAccepted: boolPtr(false), wantCode: http.StatusForbidden, wantBlockedBody: true},
		{name: "allows volunteer with accepted agreement", roles: []domain.PersonRole{{RoleType: "VOLUNTEER", IsActive: true}}, agreementAccepted: boolPtr(true), wantCode: http.StatusOK},
		{name: "ignores inactive volunteer role", roles: []domain.PersonRole{{RoleType: "VOLUNTEER", IsActive: false}}, wantCode: http.StatusOK, assertNoChecker: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := new(mockAgreementChecker)
			finder := new(mockRoleFinder)
			personID := uuid.New()

			claims := auth.AuthClaims{Subject: uuid.New().String(), Roles: []string{"VOLUNTEER"}}
			if !tt.noPerson {
				claims.PersonID = personID
				finder.On("FindByPersonID", mock.Anything, personID).Return(tt.roles, nil)
			}
			if tt.agreementAccepted != nil {
				checker.On("HasAcceptedAgreement", mock.Anything, personID).Return(*tt.agreementAccepted, nil)
			}

			ctx := auth.NewContext(context.Background(), claims)
			req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			RequireAgreement(checker, finder)(okHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantBlockedBody {
				var body map[string]string
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
				assert.Equal(t, "agreement_required", body["error"])
			}
			if tt.assertNoChecker {
				checker.AssertNotCalled(t, "HasAcceptedAgreement")
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
