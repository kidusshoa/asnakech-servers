package middleware

import (
	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/rbac"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/gin-gonic/gin"
)

// RequireRoles allows the request only if the JWT role is one of allowed.
func RequireRoles(allowed ...domain.RoleCode) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := domain.RoleCode(c.GetString(ContextRoleCode))
		if !rbac.HasAnyRole(role, allowed...) {
			response.Fail(c, apperr.Forbidden("insufficient role"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequirePermission allows the request only if the JWT role has perm.
func RequirePermission(perm rbac.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := domain.RoleCode(c.GetString(ContextRoleCode))
		if !rbac.HasPermission(role, perm) {
			response.Fail(c, apperr.Forbidden("insufficient permissions"))
			c.Abort()
			return
		}
		c.Next()
	}
}
