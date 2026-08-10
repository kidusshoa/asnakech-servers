package handlers

import (
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roles *service.RoleService
}

func NewRoleHandler(roles *service.RoleService) *RoleHandler {
	return &RoleHandler{roles: roles}
}

// ListRoles godoc
// @Summary      List platform roles
// @Description  Returns seeded system roles (student, teacher, admin, parent). Requires DATABASE_URL and applied migrations.
// @Tags         roles
// @Produce      json
// @Success      200 {object} RolesListResponse
// @Failure      500 {object} ErrorResponse
// @Router       /api/v1/roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.roles.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}

	out := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, toRoleResponse(role))
	}
	response.OK(c, out)
}

func toRoleResponse(role domain.Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID,
		Code:        string(role.Code),
		Name:        role.Name,
		Description: role.Description,
	}
}
