package rbac

import "github.com/asnakech/asnakech-servers/internal/domain"

// Permission is a fine-grained capability checked by middleware/services.
type Permission string

const (
	PermProfileRead   Permission = "profile:read"
	PermProfileWrite  Permission = "profile:write"
	PermUsersRead     Permission = "users:read"
	PermUsersManage   Permission = "users:manage"
	PermRolesRead     Permission = "roles:read"
	PermCoursesRead   Permission = "courses:read"
	PermCoursesWrite  Permission = "courses:write"
	PermCoursesManage Permission = "courses:manage"
)

var rolePermissions = map[domain.RoleCode]map[Permission]struct{}{
	domain.RoleStudent: {
		PermProfileRead:  {},
		PermProfileWrite: {},
		PermRolesRead:    {},
		PermCoursesRead:  {},
	},
	domain.RoleTeacher: {
		PermProfileRead:  {},
		PermProfileWrite: {},
		PermRolesRead:    {},
		PermCoursesRead:  {},
		PermCoursesWrite: {},
	},
	domain.RoleParent: {
		PermProfileRead:  {},
		PermProfileWrite: {},
		PermRolesRead:    {},
		PermCoursesRead:  {},
	},
	domain.RoleAdmin: {
		PermProfileRead:   {},
		PermProfileWrite:  {},
		PermUsersRead:     {},
		PermUsersManage:   {},
		PermRolesRead:     {},
		PermCoursesRead:   {},
		PermCoursesWrite:  {},
		PermCoursesManage: {},
	},
}

// HasPermission reports whether role includes perm.
func HasPermission(role domain.RoleCode, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = perms[perm]
	return ok
}

// HasAnyRole reports whether role is one of allowed.
func HasAnyRole(role domain.RoleCode, allowed ...domain.RoleCode) bool {
	for _, a := range allowed {
		if role == a {
			return true
		}
	}
	return false
}

// PermissionsFor returns a sorted-stable list of permissions for a role.
func PermissionsFor(role domain.RoleCode) []Permission {
	perms := rolePermissions[role]
	out := make([]Permission, 0, len(perms))
	for p := range perms {
		out = append(out, p)
	}
	// stable order for docs/tests
	order := []Permission{
		PermProfileRead, PermProfileWrite,
		PermUsersRead, PermUsersManage,
		PermRolesRead,
		PermCoursesRead, PermCoursesWrite, PermCoursesManage,
	}
	sorted := make([]Permission, 0, len(out))
	seen := map[Permission]struct{}{}
	for _, p := range order {
		if _, ok := perms[p]; ok {
			sorted = append(sorted, p)
			seen[p] = struct{}{}
		}
	}
	for _, p := range out {
		if _, ok := seen[p]; !ok {
			sorted = append(sorted, p)
		}
	}
	return sorted
}
