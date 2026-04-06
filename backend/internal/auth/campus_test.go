package auth

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCampusID(t *testing.T) {
	campusA := uuid.New()
	campusB := uuid.New()

	tests := []struct {
		name             string
		claims           AuthClaims
		overrideCampusID *uuid.UUID
		want             uuid.UUID
		wantErr          string
	}{
		{
			name:             "non-admin uses own campus",
			claims:           AuthClaims{CampusID: campusA, Roles: []string{"VOLUNTEER"}},
			overrideCampusID: nil,
			want:             campusA,
		},
		{
			name:             "admin with nil override uses own",
			claims:           AuthClaims{CampusID: campusA, Roles: []string{"ADMIN"}},
			overrideCampusID: nil,
			want:             campusA,
		},
		{
			name:             "admin overrides with different campus",
			claims:           AuthClaims{CampusID: campusA, Roles: []string{"ADMIN"}},
			overrideCampusID: &campusB,
			want:             campusB,
		},
		{
			name:             "admin override same as own",
			claims:           AuthClaims{CampusID: campusA, Roles: []string{"ADMIN"}},
			overrideCampusID: &campusA,
			want:             campusA,
		},
		{
			name:             "non-admin override attempt rejected",
			claims:           AuthClaims{CampusID: campusA, Roles: []string{"VOLUNTEER"}},
			overrideCampusID: &campusB,
			wantErr:          "ADMIN role",
		},
		{
			name:             "nil campus_id in claims",
			claims:           AuthClaims{CampusID: uuid.Nil, Roles: []string{"ADMIN"}},
			overrideCampusID: nil,
			wantErr:          "missing campus_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveCampusID(tt.claims, tt.overrideCampusID)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Equal(t, uuid.Nil, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
