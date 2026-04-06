package domain

// Role constants matching Keycloak realm roles.
const (
	RoleAdmin        = "ADMIN"
	RoleCoordinator  = "COORDINATOR"
	RoleProfessional = "PROFESSIONAL"
	RoleSecretary    = "SECRETARY"
	RoleVolunteer    = "VOLUNTEER"
)

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
