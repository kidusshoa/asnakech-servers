package handlers

import (
	"strconv"
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type EnrollmentHandler struct {
	enrollments *service.EnrollmentService
}

func NewEnrollmentHandler(enrollments *service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{enrollments: enrollments}
}

type enrollRequest struct {
	InviteCode string `json:"invite_code"`
}

type enrollmentSettingsRequest struct {
	EnrollmentCapacity *int  `json:"enrollment_capacity"`
	EnrollmentOpen     *bool `json:"enrollment_open"`
	WaitlistEnabled    *bool `json:"waitlist_enabled"`
}

type createInviteCodeRequest struct {
	Code      string     `json:"code"`
	MaxUses   *int       `json:"max_uses"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type EnrollmentResponse struct {
	ID           string     `json:"id"`
	CourseID     string     `json:"course_id"`
	UserID       string     `json:"user_id"`
	Status       string     `json:"status"`
	Source       string     `json:"source"`
	InviteCodeID *string    `json:"invite_code_id,omitempty"`
	EnrolledAt   *time.Time `json:"enrolled_at,omitempty"`
	WaitlistedAt *time.Time `json:"waitlisted_at,omitempty"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	UserEmail    string     `json:"user_email,omitempty"`
	UserFullName string     `json:"user_full_name,omitempty"`
	CourseTitle  string     `json:"course_title,omitempty"`
	CourseSlug   string     `json:"course_slug,omitempty"`
}

type EnrollmentEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    EnrollmentResponse `json:"data"`
}

type EnrollmentListEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []EnrollmentResponse `json:"data"`
	Meta    response.Meta        `json:"meta"`
}

type InviteCodeResponse struct {
	ID        string     `json:"id"`
	CourseID  string     `json:"course_id"`
	Code      string     `json:"code"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	UsesCount int        `json:"uses_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedBy string     `json:"created_by"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type InviteCodeEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    InviteCodeResponse `json:"data"`
}

type InviteCodeListEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []InviteCodeResponse `json:"data"`
}

type CourseAccessResponse struct {
	CourseID         string              `json:"course_id"`
	CanAccessContent bool                `json:"can_access_content"`
	IsTeacher        bool                `json:"is_teacher"`
	IsPlatformAdmin  bool                `json:"is_platform_admin"`
	Enrollment       *EnrollmentResponse `json:"enrollment,omitempty"`
}

type CourseAccessEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    CourseAccessResponse `json:"data"`
}

// Enroll godoc
// @Summary      Enroll in a course
// @Tags         enrollments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body enrollRequest false "Optional invite code"
// @Success      201 {object} EnrollmentEnvelope
// @Router       /api/v1/courses/{id}/enroll [post]
func (h *EnrollmentHandler) Enroll(c *gin.Context) {
	var req enrollRequest
	_ = c.ShouldBindJSON(&req)
	en, err := h.enrollments.Enroll(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.InviteCode,
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toEnrollmentResponse(en))
}

// Unenroll godoc
// @Summary      Unenroll from a course
// @Tags         enrollments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} EnrollmentEnvelope
// @Router       /api/v1/courses/{id}/enroll [delete]
func (h *EnrollmentHandler) Unenroll(c *gin.Context) {
	en, err := h.enrollments.Unenroll(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toEnrollmentResponse(en))
}

// ListMyEnrollments godoc
// @Summary      List my enrollments
// @Tags         enrollments
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page"
// @Param        per_page query int false "Per page"
// @Param        status query string false "active|waitlisted|cancelled"
// @Success      200 {object} EnrollmentListEnvelope
// @Router       /api/v1/me/enrollments [get]
func (h *EnrollmentHandler) ListMyEnrollments(c *gin.Context) {
	filter := enrollmentFilterFromQuery(c)
	items, total, err := h.enrollments.ListMine(c.Request.Context(), c.GetString(middleware.ContextUserID), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]EnrollmentResponse, 0, len(items))
	for i := range items {
		out = append(out, toEnrollmentResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      filter.Page,
		PerPage:   filter.PerPage,
		Total:     total,
	})
}

// ListCourseEnrollments godoc
// @Summary      List enrollments for a course
// @Tags         enrollments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        page query int false "Page"
// @Param        per_page query int false "Per page"
// @Param        status query string false "active|waitlisted|cancelled"
// @Success      200 {object} EnrollmentListEnvelope
// @Router       /api/v1/courses/{id}/enrollments [get]
func (h *EnrollmentHandler) ListCourseEnrollments(c *gin.Context) {
	filter := enrollmentFilterFromQuery(c)
	items, total, err := h.enrollments.ListForCourse(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		filter,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]EnrollmentResponse, 0, len(items))
	for i := range items {
		out = append(out, toEnrollmentResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      filter.Page,
		PerPage:   filter.PerPage,
		Total:     total,
	})
}

// GetCourseAccess godoc
// @Summary      Check course content access
// @Tags         enrollments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} CourseAccessEnvelope
// @Router       /api/v1/courses/{id}/access [get]
func (h *EnrollmentHandler) GetCourseAccess(c *gin.Context) {
	access, err := h.enrollments.GetAccess(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	resp := CourseAccessResponse{
		CourseID:         access.CourseID,
		CanAccessContent: access.CanAccessContent,
		IsTeacher:        access.IsTeacher,
		IsPlatformAdmin:  access.IsPlatformAdmin,
	}
	if access.Enrollment != nil {
		er := toEnrollmentResponse(access.Enrollment)
		resp.Enrollment = &er
	}
	response.OK(c, resp)
}

// UpdateEnrollmentSettings godoc
// @Summary      Update course enrollment settings
// @Tags         enrollments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body enrollmentSettingsRequest true "Settings"
// @Success      200 {object} CourseEnvelope
// @Router       /api/v1/courses/{id}/enrollment-settings [patch]
func (h *EnrollmentHandler) UpdateEnrollmentSettings(c *gin.Context) {
	var req enrollmentSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	course, err := h.enrollments.UpdateSettings(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.EnrollmentCapacity,
		req.EnrollmentOpen,
		req.WaitlistEnabled,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCourseResponse(course))
}

// CreateInviteCode godoc
// @Summary      Create enrollment invite code
// @Tags         enrollments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body createInviteCodeRequest false "Invite"
// @Success      201 {object} InviteCodeEnvelope
// @Router       /api/v1/courses/{id}/invite-codes [post]
func (h *EnrollmentHandler) CreateInviteCode(c *gin.Context) {
	var req createInviteCodeRequest
	_ = c.ShouldBindJSON(&req)
	inv, err := h.enrollments.CreateInviteCode(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.Code,
		req.MaxUses,
		req.ExpiresAt,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toInviteCodeResponse(inv))
}

// ListInviteCodes godoc
// @Summary      List enrollment invite codes
// @Tags         enrollments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} InviteCodeListEnvelope
// @Router       /api/v1/courses/{id}/invite-codes [get]
func (h *EnrollmentHandler) ListInviteCodes(c *gin.Context) {
	items, err := h.enrollments.ListInviteCodes(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]InviteCodeResponse, 0, len(items))
	for i := range items {
		out = append(out, toInviteCodeResponse(&items[i]))
	}
	response.OK(c, out)
}

// RevokeInviteCode godoc
// @Summary      Revoke enrollment invite code
// @Tags         enrollments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        codeId path string true "Invite code ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/courses/{id}/invite-codes/{codeId} [delete]
func (h *EnrollmentHandler) RevokeInviteCode(c *gin.Context) {
	if err := h.enrollments.RevokeInviteCode(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		c.Param("codeId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "invite code revoked"})
}

func enrollmentFilterFromQuery(c *gin.Context) domain.EnrollmentListFilter {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	return domain.EnrollmentListFilter{
		Page:    page,
		PerPage: perPage,
		Status:  domain.EnrollmentStatus(c.Query("status")),
	}
}

func toEnrollmentResponse(e *domain.Enrollment) EnrollmentResponse {
	return EnrollmentResponse{
		ID:           e.ID,
		CourseID:     e.CourseID,
		UserID:       e.UserID,
		Status:       string(e.Status),
		Source:       string(e.Source),
		InviteCodeID: e.InviteCodeID,
		EnrolledAt:   e.EnrolledAt,
		WaitlistedAt: e.WaitlistedAt,
		CancelledAt:  e.CancelledAt,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		UserEmail:    e.UserEmail,
		UserFullName: e.UserFullName,
		CourseTitle:  e.CourseTitle,
		CourseSlug:   e.CourseSlug,
	}
}

func toInviteCodeResponse(c *domain.EnrollmentInviteCode) InviteCodeResponse {
	return InviteCodeResponse{
		ID:        c.ID,
		CourseID:  c.CourseID,
		Code:      c.Code,
		MaxUses:   c.MaxUses,
		UsesCount: c.UsesCount,
		ExpiresAt: c.ExpiresAt,
		CreatedBy: c.CreatedBy,
		RevokedAt: c.RevokedAt,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
