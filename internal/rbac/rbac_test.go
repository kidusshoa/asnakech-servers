package rbac_test

import (
	"testing"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/rbac"
)

func TestAdminCanManageUsers(t *testing.T) {
	if !rbac.HasPermission(domain.RoleAdmin, rbac.PermUsersManage) {
		t.Fatal("admin should manage users")
	}
	if rbac.HasPermission(domain.RoleStudent, rbac.PermUsersManage) {
		t.Fatal("student must not manage users")
	}
}

func TestTeacherCanWriteCourses(t *testing.T) {
	if !rbac.HasPermission(domain.RoleTeacher, rbac.PermCoursesWrite) {
		t.Fatal("teacher should write courses")
	}
	if rbac.HasPermission(domain.RoleStudent, rbac.PermCoursesWrite) {
		t.Fatal("student must not write courses")
	}
}

func TestHasAnyRole(t *testing.T) {
	if !rbac.HasAnyRole(domain.RoleAdmin, domain.RoleAdmin, domain.RoleTeacher) {
		t.Fatal("expected match")
	}
	if rbac.HasAnyRole(domain.RoleStudent, domain.RoleAdmin) {
		t.Fatal("expected no match")
	}
}
