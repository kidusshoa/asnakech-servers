package handlers

import (
	"strconv"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

type updateProfileRequest struct {
	FullName *string `json:"full_name"`
	Bio      *string `json:"bio"`
	Phone    *string `json:"phone"`
	Locale   *string `json:"locale"`
	Timezone *string `json:"timezone"`
}

type setAvatarRequest struct {
	AvatarURL string `json:"avatar_url" binding:"required" example:"https://cdn.example.com/a.png"`
}

type adminUpdateUserRequest struct {
	FullName *string `json:"full_name"`
	Role     *string `json:"role" example:"teacher"`
	IsActive *bool   `json:"is_active"`
}

type UserListResponse struct {
	Success bool         `json:"success" example:"true"`
	Data    []UserPublic `json:"data"`
	Meta    response.Meta `json:"meta"`
}

type AvatarUploadIntent struct {
	Method    string `json:"method" example:"PUT"`
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Note      string `json:"note"`
}

type AvatarIntentResponse struct {
	Success bool               `json:"success" example:"true"`
	Data    AvatarUploadIntent `json:"data"`
}

// GetMyProfile godoc
// @Summary      Get my profile
// @Description  Return the authenticated user's profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} MeResponse
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/users/me [get]
func (h *UserHandler) GetMyProfile(c *gin.Context) {
	user, err := h.users.GetByID(c.Request.Context(), c.GetString(middleware.ContextUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserPublic(user))
}

// UpdateMyProfile godoc
// @Summary      Update my profile
// @Description  Patch profile fields for the authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body updateProfileRequest true "Profile fields"
// @Success      200 {object} MeResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/users/me [patch]
func (h *UserHandler) UpdateMyProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	user, err := h.users.UpdateProfile(c.Request.Context(), c.GetString(middleware.ContextUserID), domain.UserProfileUpdate{
		FullName: req.FullName,
		Bio:      req.Bio,
		Phone:    req.Phone,
		Locale:   req.Locale,
		Timezone: req.Timezone,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserPublic(user))
}

// SetMyAvatar godoc
// @Summary      Set my avatar URL
// @Description  Temporary avatar hook until Stage 14 presigned uploads
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body setAvatarRequest true "Avatar URL"
// @Success      200 {object} MeResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/users/me/avatar [put]
func (h *UserHandler) SetMyAvatar(c *gin.Context) {
	var req setAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	user, err := h.users.SetAvatarURL(c.Request.Context(), c.GetString(middleware.ContextUserID), req.AvatarURL)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserPublic(user))
}

// AvatarUploadIntent godoc
// @Summary      Avatar upload intent
// @Description  Placeholder for future presigned upload flow
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} AvatarIntentResponse
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/users/me/avatar/upload-intent [get]
func (h *UserHandler) AvatarUploadIntent(c *gin.Context) {
	intent := h.users.AvatarUploadIntent(c.GetString(middleware.ContextUserID))
	response.OK(c, AvatarUploadIntent{
		Method:    intent.Method,
		UploadURL: intent.UploadURL,
		PublicURL: intent.PublicURL,
		Note:      intent.Note,
	})
}

// AdminListUsers godoc
// @Summary      List users (admin)
// @Description  Paginated user directory for administrators
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page" default(1)
// @Param        per_page query int false "Page size" default(20)
// @Param        role query string false "Filter by role code"
// @Param        q query string false "Search email or name"
// @Success      200 {object} UserListResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /api/v1/admin/users [get]
func (h *UserHandler) AdminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	users, total, filter, err := h.users.List(c.Request.Context(), domain.UserListFilter{
		Role:    domain.RoleCode(c.Query("role")),
		Query:   c.Query("q"),
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	out := make([]UserPublic, 0, len(users))
	for i := range users {
		out = append(out, toUserPublic(&users[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      filter.Page,
		PerPage:   filter.PerPage,
		Total:     total,
	})
}

// AdminGetUser godoc
// @Summary      Get user (admin)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} MeResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/admin/users/{id} [get]
func (h *UserHandler) AdminGetUser(c *gin.Context) {
	user, err := h.users.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserPublic(user))
}

// AdminUpdateUser godoc
// @Summary      Update user (admin)
// @Description  Change role, active flag, or name
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        body body adminUpdateUserRequest true "Admin patch"
// @Success      200 {object} MeResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/admin/users/{id} [patch]
func (h *UserHandler) AdminUpdateUser(c *gin.Context) {
	var req adminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}

	patch := domain.AdminUserUpdate{
		FullName: req.FullName,
		IsActive: req.IsActive,
	}
	if req.Role != nil {
		code := domain.RoleCode(*req.Role)
		switch code {
		case domain.RoleStudent, domain.RoleTeacher, domain.RoleAdmin, domain.RoleParent:
			patch.RoleCode = &code
		default:
			response.Fail(c, apperr.Validation("invalid role"))
			return
		}
	}

	user, err := h.users.AdminUpdate(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), patch)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserPublic(user))
}

// AdminDeleteUser godoc
// @Summary      Soft-delete user (admin)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/admin/users/{id} [delete]
func (h *UserHandler) AdminDeleteUser(c *gin.Context) {
	if err := h.users.SoftDelete(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "user deleted"})
}
