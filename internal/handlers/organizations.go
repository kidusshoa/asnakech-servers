package handlers

import (
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct {
	orgs *service.OrganizationService
}

func NewOrganizationHandler(orgs *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{orgs: orgs}
}

type createOrgRequest struct {
	Name        string `json:"name" binding:"required" example:"Asnakech Academy"`
	Slug        string `json:"slug" example:"asnakech-academy"`
	Description string `json:"description"`
}

type updateOrgRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	LogoURL     *string `json:"logo_url"`
}

type createInviteRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" example:"member"`
}

type acceptInviteRequest struct {
	Token string `json:"token" binding:"required"`
}

type updateMemberRoleRequest struct {
	Role string `json:"role" binding:"required" example:"admin"`
}

type OrganizationResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	LogoURL     string    `json:"logo_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrganizationEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    OrganizationResponse `json:"data"`
}

type OrganizationListEnvelope struct {
	Success bool                   `json:"success" example:"true"`
	Data    []OrganizationResponse `json:"data"`
}

type MemberResponse struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	OrgRole   string    `json:"org_role"`
	JoinedAt  time.Time `json:"joined_at"`
}

type MemberListEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    []MemberResponse `json:"data"`
}

type InviteResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	OrgRole   string    `json:"org_role"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	TokenDev  string    `json:"token,omitempty"`
}

type InviteEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    InviteResponse `json:"data"`
}

type InviteListEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    []InviteResponse `json:"data"`
}

// CreateOrganization godoc
// @Summary      Create organization
// @Description  Create a school/org; caller becomes owner
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createOrgRequest true "Organization"
// @Success      201 {object} OrganizationEnvelope
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /api/v1/organizations [post]
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	var req createOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	org, err := h.orgs.Create(c.Request.Context(), c.GetString(middleware.ContextUserID), service.CreateOrganizationInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toOrgResponse(org))
}

// ListMyOrganizations godoc
// @Summary      List my organizations
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} OrganizationListEnvelope
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/organizations [get]
func (h *OrganizationHandler) ListMyOrganizations(c *gin.Context) {
	orgs, err := h.orgs.ListMine(c.Request.Context(), c.GetString(middleware.ContextUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]OrganizationResponse, 0, len(orgs))
	for i := range orgs {
		out = append(out, toOrgResponse(&orgs[i]))
	}
	response.OK(c, out)
}

// GetOrganization godoc
// @Summary      Get organization
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Success      200 {object} OrganizationEnvelope
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/organizations/{id} [get]
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	org, _, err := h.orgs.Get(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrgResponse(org))
}

// UpdateOrganization godoc
// @Summary      Update organization
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Param        body body updateOrgRequest true "Patch"
// @Success      200 {object} OrganizationEnvelope
// @Failure      400 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /api/v1/organizations/{id} [patch]
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	var req updateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	org, err := h.orgs.Update(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), domain.OrganizationUpdate{
		Name:        req.Name,
		Description: req.Description,
		LogoURL:     req.LogoURL,
	}, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrgResponse(org))
}

// DeleteOrganization godoc
// @Summary      Delete organization
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Success      200 {object} MessageResponse
// @Failure      403 {object} ErrorResponse
// @Router       /api/v1/organizations/{id} [delete]
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	if err := h.orgs.Delete(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "organization deleted"})
}

// ListMembers godoc
// @Summary      List organization members
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Success      200 {object} MemberListEnvelope
// @Router       /api/v1/organizations/{id}/members [get]
func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	members, err := h.orgs.ListMembers(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]MemberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, MemberResponse{
			UserID:   m.UserID,
			Email:    m.UserEmail,
			FullName: m.UserFullName,
			OrgRole:  string(m.OrgRole),
			JoinedAt: m.JoinedAt,
		})
	}
	response.OK(c, out)
}

// UpdateMemberRole godoc
// @Summary      Update member role
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Param        userId path string true "User ID"
// @Param        body body updateMemberRoleRequest true "Role"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/organizations/{id}/members/{userId} [patch]
func (h *OrganizationHandler) UpdateMemberRole(c *gin.Context) {
	var req updateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	if err := h.orgs.UpdateMemberRole(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		c.Param("userId"),
		domain.OrgRole(req.Role),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "member role updated"})
}

// RemoveMember godoc
// @Summary      Remove member
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Param        userId path string true "User ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/organizations/{id}/members/{userId} [delete]
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	if err := h.orgs.RemoveMember(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		c.Param("userId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "member removed"})
}

// CreateInvite godoc
// @Summary      Invite member
// @Description  In development, response may include token for accepting without email
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Param        body body createInviteRequest true "Invite"
// @Success      201 {object} InviteEnvelope
// @Router       /api/v1/organizations/{id}/invites [post]
func (h *OrganizationHandler) CreateInvite(c *gin.Context) {
	var req createInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	role := domain.OrgRoleMember
	if req.Role != "" {
		role = domain.OrgRole(req.Role)
	}
	result, err := h.orgs.CreateInvite(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.Email,
		role,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toInviteResponse(result.Invite, result.TokenDev))
}

// ListInvites godoc
// @Summary      List pending invites
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Success      200 {object} InviteListEnvelope
// @Router       /api/v1/organizations/{id}/invites [get]
func (h *OrganizationHandler) ListInvites(c *gin.Context) {
	invites, err := h.orgs.ListInvites(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]InviteResponse, 0, len(invites))
	for i := range invites {
		out = append(out, toInviteResponse(&invites[i], ""))
	}
	response.OK(c, out)
}

// RevokeInvite godoc
// @Summary      Revoke invite
// @Tags         organizations
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Organization ID"
// @Param        inviteId path string true "Invite ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/organizations/{id}/invites/{inviteId} [delete]
func (h *OrganizationHandler) RevokeInvite(c *gin.Context) {
	if err := h.orgs.RevokeInvite(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		c.Param("inviteId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "invite revoked"})
}

// AcceptInvite godoc
// @Summary      Accept organization invite
// @Tags         organizations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body acceptInviteRequest true "Invite token"
// @Success      200 {object} OrganizationEnvelope
// @Router       /api/v1/organizations/invites/accept [post]
func (h *OrganizationHandler) AcceptInvite(c *gin.Context) {
	var req acceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	org, err := h.orgs.AcceptInvite(c.Request.Context(), c.GetString(middleware.ContextUserID), req.Token)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrgResponse(org))
}

func isPlatformAdmin(c *gin.Context) bool {
	return c.GetString(middleware.ContextRoleCode) == string(domain.RoleAdmin)
}

func toOrgResponse(org *domain.Organization) OrganizationResponse {
	return OrganizationResponse{
		ID:          org.ID,
		Name:        org.Name,
		Slug:        org.Slug,
		Description: org.Description,
		LogoURL:     org.LogoURL,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}
}

func toInviteResponse(inv *domain.OrganizationInvite, tokenDev string) InviteResponse {
	return InviteResponse{
		ID:        inv.ID,
		Email:     inv.Email,
		OrgRole:   string(inv.OrgRole),
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
		TokenDev:  tokenDev,
	}
}
