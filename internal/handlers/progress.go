package handlers

import (
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type ProgressHandler struct {
	progress *service.ProgressService
}

func NewProgressHandler(progress *service.ProgressService) *ProgressHandler {
	return &ProgressHandler{progress: progress}
}

type upsertLessonProgressRequest struct {
	Percent      *int    `json:"percent"`
	LastPosition *string `json:"last_position"`
	Completed    *bool   `json:"completed"`
}

type LessonProgressResponse struct {
	ID           string     `json:"id,omitempty"`
	UserID       string     `json:"user_id"`
	CourseID     string     `json:"course_id"`
	LessonID     string     `json:"lesson_id"`
	Status       string     `json:"status"`
	Percent      int        `json:"percent"`
	LastPosition string     `json:"last_position"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	LessonTitle  string     `json:"lesson_title,omitempty"`
	LessonSlug   string     `json:"lesson_slug,omitempty"`
}

type CourseProgressResponse struct {
	ID               string     `json:"id,omitempty"`
	UserID           string     `json:"user_id"`
	CourseID         string     `json:"course_id"`
	EnrollmentID     *string    `json:"enrollment_id,omitempty"`
	Percent          int        `json:"percent"`
	CompletedLessons int        `json:"completed_lessons"`
	TotalLessons     int        `json:"total_lessons"`
	LastLessonID     *string    `json:"last_lesson_id,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CourseTitle      string     `json:"course_title,omitempty"`
	CourseSlug       string     `json:"course_slug,omitempty"`
}

type UpsertProgressData struct {
	Lesson LessonProgressResponse `json:"lesson"`
	Course CourseProgressResponse `json:"course"`
}

type UpsertProgressEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    UpsertProgressData `json:"data"`
}

type LessonProgressEnvelope struct {
	Success bool                   `json:"success" example:"true"`
	Data    LessonProgressResponse `json:"data"`
}

type CourseProgressDetailData struct {
	Course  CourseProgressResponse   `json:"course"`
	Lessons []LessonProgressResponse `json:"lessons"`
}

type CourseProgressDetailEnvelope struct {
	Success bool                     `json:"success" example:"true"`
	Data    CourseProgressDetailData `json:"data"`
}

type DashboardProgressItem struct {
	CourseProgressResponse
	EnrollmentStatus string `json:"enrollment_status,omitempty"`
}

type DashboardProgressEnvelope struct {
	Success bool                    `json:"success" example:"true"`
	Data    []DashboardProgressItem `json:"data"`
}

// UpsertLessonProgress godoc
// @Summary      Upsert lesson progress (idempotent)
// @Tags         progress
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Param        body body upsertLessonProgressRequest true "Progress"
// @Success      200 {object} UpsertProgressEnvelope
// @Router       /api/v1/lessons/{lessonId}/progress [put]
func (h *ProgressHandler) UpsertLessonProgress(c *gin.Context) {
	var req upsertLessonProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	lesson, course, err := h.progress.UpsertLessonProgress(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
		domain.LessonProgressUpsert{
			Percent:      req.Percent,
			LastPosition: req.LastPosition,
			Completed:    req.Completed,
		},
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, UpsertProgressData{
		Lesson: toLessonProgressResponse(lesson),
		Course: toCourseProgressResponse(course),
	})
}

// GetLessonProgress godoc
// @Summary      Get my progress on a lesson
// @Tags         progress
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Success      200 {object} LessonProgressEnvelope
// @Router       /api/v1/lessons/{lessonId}/progress [get]
func (h *ProgressHandler) GetLessonProgress(c *gin.Context) {
	p, err := h.progress.GetLessonProgress(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLessonProgressResponse(p))
}

// GetCourseProgress godoc
// @Summary      Get my progress on a course
// @Tags         progress
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} CourseProgressDetailEnvelope
// @Router       /api/v1/courses/{id}/progress [get]
func (h *ProgressHandler) GetCourseProgress(c *gin.Context) {
	cp, lessons, err := h.progress.GetCourseProgress(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]LessonProgressResponse, 0, len(lessons))
	for i := range lessons {
		out = append(out, toLessonProgressResponse(&lessons[i]))
	}
	response.OK(c, CourseProgressDetailData{
		Course:  toCourseProgressResponse(cp),
		Lessons: out,
	})
}

// ListMyProgress godoc
// @Summary      Student progress dashboard
// @Tags         progress
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} DashboardProgressEnvelope
// @Router       /api/v1/me/progress [get]
func (h *ProgressHandler) ListMyProgress(c *gin.Context) {
	items, err := h.progress.ListMyProgress(c.Request.Context(), c.GetString(middleware.ContextUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]DashboardProgressItem, 0, len(items))
	for i := range items {
		item := DashboardProgressItem{
			CourseProgressResponse: toCourseProgressResponse(&items[i].CourseProgress),
			EnrollmentStatus:       string(items[i].EnrollmentStatus),
		}
		out = append(out, item)
	}
	response.OK(c, out)
}

func toLessonProgressResponse(p *domain.LessonProgress) LessonProgressResponse {
	resp := LessonProgressResponse{
		ID:           p.ID,
		UserID:       p.UserID,
		CourseID:     p.CourseID,
		LessonID:     p.LessonID,
		Status:       string(p.Status),
		Percent:      p.Percent,
		LastPosition: p.LastPosition,
		CompletedAt:  p.CompletedAt,
		LessonTitle:  p.LessonTitle,
		LessonSlug:   p.LessonSlug,
	}
	if !p.CreatedAt.IsZero() {
		t := p.CreatedAt
		resp.CreatedAt = &t
	}
	if !p.UpdatedAt.IsZero() {
		t := p.UpdatedAt
		resp.UpdatedAt = &t
	}
	return resp
}

func toCourseProgressResponse(p *domain.CourseProgress) CourseProgressResponse {
	return CourseProgressResponse{
		ID:               p.ID,
		UserID:           p.UserID,
		CourseID:         p.CourseID,
		EnrollmentID:     p.EnrollmentID,
		Percent:          p.Percent,
		CompletedLessons: p.CompletedLessons,
		TotalLessons:     p.TotalLessons,
		LastLessonID:     p.LastLessonID,
		CompletedAt:      p.CompletedAt,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		CourseTitle:      p.CourseTitle,
		CourseSlug:       p.CourseSlug,
	}
}
