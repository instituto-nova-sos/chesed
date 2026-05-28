package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasRole(t *testing.T) {
	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		want          bool
	}{
		{
			name:          "single role matches",
			userRoles:     []string{"ADMIN"},
			requiredRoles: []string{"ADMIN"},
			want:          true,
		},
		{
			name:          "match one of many required",
			userRoles:     []string{"SECRETARY"},
			requiredRoles: []string{"ADMIN", "SECRETARY"},
			want:          true,
		},
		{
			name:          "match one of many user roles",
			userRoles:     []string{"VOLUNTEER", "COORDINATOR"},
			requiredRoles: []string{"COORDINATOR"},
			want:          true,
		},
		{
			name:          "no match",
			userRoles:     []string{"VOLUNTEER"},
			requiredRoles: []string{"ADMIN"},
			want:          false,
		},
		{
			name:          "empty user roles",
			userRoles:     []string{},
			requiredRoles: []string{"ADMIN"},
			want:          false,
		},
		{
			name:          "empty required roles",
			userRoles:     []string{"ADMIN"},
			requiredRoles: []string{},
			want:          false,
		},
		{
			name:          "both empty",
			userRoles:     []string{},
			requiredRoles: []string{},
			want:          false,
		},
		{
			name:          "nil user roles",
			userRoles:     nil,
			requiredRoles: []string{"ADMIN"},
			want:          false,
		},
		{
			name:          "nil required roles",
			userRoles:     []string{"ADMIN"},
			requiredRoles: nil,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasRole(tt.userRoles, tt.requiredRoles...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoleHierarchy(t *testing.T) {
	assert.Len(t, RoleHierarchy, 5, "RoleHierarchy should have exactly 5 entries")

	assert.Equal(t, 1, RoleHierarchy[RoleVolunteer])
	assert.Equal(t, 2, RoleHierarchy[RoleSecretary])
	assert.Equal(t, 3, RoleHierarchy[RoleProfessional])
	assert.Equal(t, 4, RoleHierarchy[RoleCoordinator])
	assert.Equal(t, 5, RoleHierarchy[RoleAdmin])

	// Verify strict ordering
	assert.Less(t, RoleHierarchy[RoleVolunteer], RoleHierarchy[RoleSecretary])
	assert.Less(t, RoleHierarchy[RoleSecretary], RoleHierarchy[RoleProfessional])
	assert.Less(t, RoleHierarchy[RoleProfessional], RoleHierarchy[RoleCoordinator])
	assert.Less(t, RoleHierarchy[RoleCoordinator], RoleHierarchy[RoleAdmin])
}

func TestRequiresVolunteerBase(t *testing.T) {
	tests := []struct {
		roleType string
		want     bool
	}{
		{RoleProfessional, true},
		{RoleCoordinator, true},
		{RoleAdmin, true},
		{RoleVolunteer, false},
		{RoleAssisted, false},
		{RoleSecretary, false},
		{"UNKNOWN", false},
	}

	for _, tt := range tests {
		t.Run(tt.roleType, func(t *testing.T) {
			assert.Equal(t, tt.want, RequiresVolunteerBase(tt.roleType))
		})
	}
}
