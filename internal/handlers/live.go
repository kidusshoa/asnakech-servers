package handlers

import (
	"strconv"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type LiveHandler struct {
	live *service.LiveService
}

func NewLiveHandler(live *service.LiveService) *LiveHandler {
	return &LiveHandler{live: live}
}

type createLiveSessionRequest struct {
	LessonID    *string   `json:"lesson_id"`
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	StartsAt    time.Time `json:"starts_at" binding:"required"`
	EndsAt      time.Time `json:"ends_at" binding:"required"`
	Timezone    string    `json:"timezone"`
	Provider    string    `json:"provider"`
	JoinURL     string    `json:"join_url"`
	HostURL     string    `json:"host_url"`
}

type updateLiveSessionRequest struct {
	LessonID    *string    `json:"lesson_id"`
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	Timezone    *string    `json:"timezone"`
	Provider    *string    `json:"provider"`
	JoinURL     *string    `json:"join_url"`
	HostURL     *string    `json:"host_url"`
}

type markAttendanceRequest struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

type LiveSessionResponse struct {
	ID               string            `json:"id"`
	CourseID         string            `json:"course_id"`
	LessonID         *string           `json:"lesson_id,omitempty"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           string            `json:"status"`
	StartsAt         time.Time         `json:"starts_at"`
	EndsAt           time.Time         `json:"ends_at"`
	Timezone         string            `json:"timezone"`
	Provider         string            `json:"provider"`
	JoinURL          string            `json:"join_url,omitempty"`
	HostURL          string            `json:"host_url,omitempty"`
	ExternalID       string            `json:"external_id,omitempty"`
	ProviderMetadata map[string]string `json:"provider_metadata,omitempty"`
	CreatedBy        string            `json:"created_by"`
	CourseTitle      string            `json:"course_title,omitempty"`
	CourseSlug       string            `json:"course_slug,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type LiveSessionEnvelope struct {
	Success bool                `json:"success" example:"true"`
	Data    LiveSessionResponse `json:"data"`
}

type LiveSessionListEnvelope struct {
	Success bool                  `json:"success" example:"true"`
	Data    []LiveSessionResponse `json:"data"`
	Meta    response.Meta         `json:"meta"`
}

type JoinInfoResponse struct {
	SessionID string    `json:"session_id"`
	JoinURL   string    `json:"join_url"`
	HostURL   string    `json:"host_url,omitempty"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Status    string    `json:"status"`
	IsHost    bool      `json:"is_host"`
}

type JoinInfoEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    JoinInfoResponse `json:"data"`
}

type AttendanceResponse struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"session_id"`
	UserID       string     `json:"user_id"`
	Status       string     `json:"status"`
	JoinedAt     *time.Time `json:"joined_at,omitempty"`
	LeftAt       *time.Time `json:"left_at,omitempty"`
	MarkedBy     *string    `json:"marked_by,omitempty"`
	Note         string     `json:"note,omitempty"`
	UserEmail    string     `json:"user_email,omitempty"`
	UserFullName string     `json:"user_full_name,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AttendanceEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    AttendanceResponse `json:"data"`
}

type AttendanceListEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []AttendanceResponse `json:"data"`
	Meta    response.Meta        `json:"meta"`
}

// CreateSession godoc
// @Summary      Create live class session
// @Tags         live
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body createLiveSessionRequest true "Session"
// @Success      201 {object} LiveSessionEnvelope
// @Router       /api/v1/courses/{id}/sessions [post]
func (h *LiveHandler) CreateSession(c *gin.Context) {
	var req createLiveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	var provider domain.LiveProvider
	if req.Provider != "" {
		provider = domain.LiveProvider(req.Provider)
	}
	session, err := h.live.CreateSession(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), domain.LiveSessionCreate{
		LessonID:    req.LessonID,
		Title:       req.Title,
		Description: req.Description,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		Timezone:    req.Timezone,
		Provider:    provider,
		JoinURL:     req.JoinURL,
		HostURL:     req.HostURL,
	}, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toLiveSessionResponse(session))
}

// ListCourseSessions godoc
// @Summary      List course live sessions
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} LiveSessionListEnvelope
// @Router       /api/v1/courses/{id}/sessions [get]
func (h *LiveHandler) ListCourseSessions(c *gin.Context) {
	items, err := h.live.ListCourseSessions(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]LiveSessionResponse, 0, len(items))
	for i := range items {
		out = append(out, toLiveSessionResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Total:     int64(len(out)),
	})
}

// GetSession godoc
// @Summary      Get live session
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} LiveSessionEnvelope
// @Router       /api/v1/sessions/{sessionId} [get]
func (h *LiveHandler) GetSession(c *gin.Context) {
	session, err := h.live.GetSession(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLiveSessionResponse(session))
}

// UpdateSession godoc
// @Summary      Update live session
// @Tags         live
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Param        body body updateLiveSessionRequest true "Patch"
// @Success      200 {object} LiveSessionEnvelope
// @Router       /api/v1/sessions/{sessionId} [patch]
func (h *LiveHandler) UpdateSession(c *gin.Context) {
	var req updateLiveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	var provider *domain.LiveProvider
	if req.Provider != nil {
		p := domain.LiveProvider(*req.Provider)
		provider = &p
	}
	session, err := h.live.UpdateSession(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), domain.LiveSessionUpdate{
		LessonID:    req.LessonID,
		Title:       req.Title,
		Description: req.Description,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		Timezone:    req.Timezone,
		Provider:    provider,
		JoinURL:     req.JoinURL,
		HostURL:     req.HostURL,
	}, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLiveSessionResponse(session))
}

// PublishSession godoc
// @Summary      Publish live session to students
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} LiveSessionEnvelope
// @Router       /api/v1/sessions/{sessionId}/publish [post]
func (h *LiveHandler) PublishSession(c *gin.Context) {
	session, err := h.live.PublishSession(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLiveSessionResponse(session))
}

// CompleteSession godoc
// @Summary      Mark live session completed
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} LiveSessionEnvelope
// @Router       /api/v1/sessions/{sessionId}/complete [post]
func (h *LiveHandler) CompleteSession(c *gin.Context) {
	session, err := h.live.CompleteSession(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLiveSessionResponse(session))
}

// CancelSession godoc
// @Summary      Cancel live session
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} LiveSessionEnvelope
// @Router       /api/v1/sessions/{sessionId}/cancel [post]
func (h *LiveHandler) CancelSession(c *gin.Context) {
	session, err := h.live.CancelSession(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLiveSessionResponse(session))
}

// GenerateLink godoc
// @Summary      Generate join link via provider adapter
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} LiveSessionEnvelope
// @Router       /api/v1/sessions/{sessionId}/generate-link [post]
func (h *LiveHandler) GenerateLink(c *gin.Context) {
	session, err := h.live.GenerateLink(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLiveSessionResponse(session))
}

// JoinSession godoc
// @Summary      Get join link for a session
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} JoinInfoEnvelope
// @Router       /api/v1/sessions/{sessionId}/join [get]
func (h *LiveHandler) JoinSession(c *gin.Context) {
	info, err := h.live.JoinSession(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, JoinInfoResponse{
		SessionID: info.SessionID,
		JoinURL:   info.JoinURL,
		HostURL:   info.HostURL,
		StartsAt:  info.StartsAt,
		EndsAt:    info.EndsAt,
		Status:    string(info.Status),
		IsHost:    info.IsHost,
	})
}

// ListAttendance godoc
// @Summary      List session attendance
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} AttendanceListEnvelope
// @Router       /api/v1/sessions/{sessionId}/attendance [get]
func (h *LiveHandler) ListAttendance(c *gin.Context) {
	items, err := h.live.ListAttendance(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]AttendanceResponse, 0, len(items))
	for i := range items {
		out = append(out, toAttendanceResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Total:     int64(len(out)),
	})
}

// MarkAttendance godoc
// @Summary      Mark user attendance
// @Tags         live
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Param        userId path string true "User ID"
// @Param        body body markAttendanceRequest true "Status"
// @Success      200 {object} AttendanceEnvelope
// @Router       /api/v1/sessions/{sessionId}/attendance/{userId} [put]
func (h *LiveHandler) MarkAttendance(c *gin.Context) {
	var req markAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	a, err := h.live.MarkAttendance(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("sessionId"),
		c.Param("userId"),
		domain.AttendanceStatus(req.Status),
		req.Note,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAttendanceResponse(a))
}

// CheckIn godoc
// @Summary      Self check-in to live session
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} AttendanceEnvelope
// @Router       /api/v1/sessions/{sessionId}/attendance/check-in [post]
func (h *LiveHandler) CheckIn(c *gin.Context) {
	a, err := h.live.CheckIn(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("sessionId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAttendanceResponse(a))
}

// ListCalendar godoc
// @Summary      Calendar feed for teacher/student sessions
// @Tags         live
// @Produce      json
// @Security     BearerAuth
// @Param        from query string true "RFC3339 start"
// @Param        to query string true "RFC3339 end"
// @Param        page query int false "Page"
// @Param        per_page query int false "Per page"
// @Success      200 {object} LiveSessionListEnvelope
// @Router       /api/v1/me/calendar [get]
func (h *LiveHandler) ListCalendar(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		response.Fail(c, apperr.Validation("from and to query params are required (RFC3339)"))
		return
	}
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		response.Fail(c, apperr.Validation("invalid from timestamp"))
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		response.Fail(c, apperr.Validation("invalid to timestamp"))
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	items, total, err := h.live.ListCalendar(c.Request.Context(), c.GetString(middleware.ContextUserID), from, to, page, perPage, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]LiveSessionResponse, 0, len(items))
	for i := range items {
		out = append(out, toLiveSessionResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      page,
		PerPage:   perPage,
		Total:     int64(total),
	})
}

func toLiveSessionResponse(s *domain.LiveSession) LiveSessionResponse {
	resp := LiveSessionResponse{
		ID:               s.ID,
		CourseID:         s.CourseID,
		LessonID:         s.LessonID,
		Title:            s.Title,
		Description:      s.Description,
		Status:           string(s.Status),
		StartsAt:         s.StartsAt,
		EndsAt:           s.EndsAt,
		Timezone:         s.Timezone,
		Provider:         string(s.Provider),
		JoinURL:          s.JoinURL,
		HostURL:          s.HostURL,
		ExternalID:       s.ExternalID,
		ProviderMetadata: s.ProviderMetadata,
		CreatedBy:        s.CreatedBy,
		CourseTitle:      s.CourseTitle,
		CourseSlug:       s.CourseSlug,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
	return resp
}

func toAttendanceResponse(a *domain.SessionAttendance) AttendanceResponse {
	return AttendanceResponse{
		ID:           a.ID,
		SessionID:    a.SessionID,
		UserID:       a.UserID,
		Status:       string(a.Status),
		JoinedAt:     a.JoinedAt,
		LeftAt:       a.LeftAt,
		MarkedBy:     a.MarkedBy,
		Note:         a.Note,
		UserEmail:    a.UserEmail,
		UserFullName: a.UserFullName,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}
