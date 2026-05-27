package domain

// Role constants matching Keycloak realm roles.
const (
	RoleAdmin        = "ADMIN"
	RoleCoordinator  = "COORDINATOR"
	RoleProfessional = "PROFESSIONAL"
	RoleSecretary    = "SECRETARY"
	RoleVolunteer    = "VOLUNTEER"
	RoleAssisted     = "ASSISTED"
)

// rolesRequiringVolunteerBase lists person roles that auto-require VOLUNTEER.
// ASSISTED and VOLUNTEER itself are excluded.
var rolesRequiringVolunteerBase = map[string]bool{
	RoleProfessional: true,
	RoleCoordinator:  true,
	RoleAdmin:        true,
}

// RequiresVolunteerBase returns true if the given person role type
// requires VOLUNTEER to be assigned as a base role.
func RequiresVolunteerBase(roleType string) bool {
	return rolesRequiringVolunteerBase[roleType]
}

// RoleHierarchy defines the privilege level of each role.
// Higher number = more permissions.
var RoleHierarchy = map[string]int{
	RoleVolunteer:    1,
	RoleSecretary:    2,
	RoleProfessional: 3,
	RoleCoordinator:  4,
	RoleAdmin:        5,
}

// HasRole checks if the user has at least one of the required roles.
func HasRole(userRoles []string, requiredRoles ...string) bool {
	for _, userRole := range userRoles {
		for _, required := range requiredRoles {
			if userRole == required {
				return true
			}
		}
	}
	return false
}
