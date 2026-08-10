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

type roleResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListRoles godoc
// @Summary List platform roles
// @Description Returns seeded system roles (student, teacher, admin, parent)
// @Tags roles
// @Produce json
// @Success 200 {object} response.Envelope
// @Failure 500 {object} response.Envelope
// @Router /roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.roles.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}

	out := make([]roleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, toRoleResponse(role))
	}
	response.OK(c, out)
}

func toRoleResponse(role domain.Role) roleResponse {
	return roleResponse{
		ID:          role.ID,
		Code:        string(role.Code),
		Name:        role.Name,
		Description: role.Description,
	}
}
