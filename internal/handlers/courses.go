package handlers

import (
	"strconv"
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/rbac"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	courses *service.CourseService
}

func NewCourseHandler(courses *service.CourseService) *CourseHandler {
	return &CourseHandler{courses: courses}
}

type createCourseRequest struct {
	OrganizationID *string  `json:"organization_id"`
	CategoryID     *string  `json:"category_id"`
	Title          string   `json:"title" binding:"required"`
	Slug           string   `json:"slug"`
	Summary        string   `json:"summary"`
	Description    string   `json:"description"`
	CoverURL       string   `json:"cover_url"`
	PriceCents     int      `json:"price_cents"`
	Currency       string   `json:"currency" example:"ETB"`
	Level          string   `json:"level" example:"beginner"`
	Language       string   `json:"language" example:"en"`
	Tags           []string `json:"tags"`
}

type updateCourseRequest struct {
	CategoryID  *string `json:"category_id"`
	Title       *string `json:"title"`
	Summary     *string `json:"summary"`
	Description *string `json:"description"`
	CoverURL    *string `json:"cover_url"`
	PriceCents  *int    `json:"price_cents"`
	Currency    *string `json:"currency"`
	Level       *string `json:"level"`
	Language    *string `json:"language"`
}

type setTagsRequest struct {
	Tags []string `json:"tags" binding:"required"`
}

type createCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type CourseResponse struct {
	ID                 string     `json:"id"`
	OrganizationID     *string    `json:"organization_id,omitempty"`
	TeacherID          string     `json:"teacher_id"`
	TeacherName        string     `json:"teacher_name"`
	CategoryID         *string    `json:"category_id,omitempty"`
	CategoryName       string     `json:"category_name,omitempty"`
	Title              string     `json:"title"`
	Slug               string     `json:"slug"`
	Summary            string     `json:"summary"`
	Description        string     `json:"description"`
	Status             string     `json:"status"`
	CoverURL           string     `json:"cover_url"`
	PriceCents         int        `json:"price_cents"`
	Currency           string     `json:"currency"`
	Level              string     `json:"level"`
	Language           string     `json:"language"`
	Tags               []string   `json:"tags"`
	EnrollmentCapacity *int       `json:"enrollment_capacity,omitempty"`
	EnrollmentOpen     bool       `json:"enrollment_open"`
	WaitlistEnabled    bool       `json:"waitlist_enabled"`
	PublishedAt        *time.Time `json:"published_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CourseEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    CourseResponse `json:"data"`
}

type CourseListEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    []CourseResponse `json:"data"`
	Meta    response.Meta    `json:"meta"`
}

type CategoryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CategoryListEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    []CategoryResponse `json:"data"`
}

type CategoryEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    CategoryResponse `json:"data"`
}

// ListCategories godoc
// @Summary      List categories
// @Tags         courses
// @Produce      json
// @Success      200 {object} CategoryListEnvelope
// @Router       /api/v1/categories [get]
func (h *CourseHandler) ListCategories(c *gin.Context) {
	cats, err := h.courses.ListCategories(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]CategoryResponse, 0, len(cats))
	for _, cat := range cats {
		out = append(out, toCategoryResponse(cat))
	}
	response.OK(c, out)
}

// CreateCategory godoc
// @Summary      Create category
// @Tags         courses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createCategoryRequest true "Category"
// @Success      201 {object} CategoryEnvelope
// @Router       /api/v1/categories [post]
func (h *CourseHandler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	cat, err := h.courses.CreateCategory(c.Request.Context(), req.Name, req.Slug, req.Description)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toCategoryResponse(*cat))
}

// ListCourses godoc
// @Summary      List courses
// @Description  Public catalog defaults to published. Teachers/admins may see drafts they own (or all if admin).
// @Tags         courses
// @Produce      json
// @Param        page query int false "Page"
// @Param        per_page query int false "Page size"
// @Param        q query string false "Search"
// @Param        category query string false "Category slug"
// @Param        tag query string false "Tag slug"
// @Param        organization_id query string false "Organization ID"
// @Param        teacher_id query string false "Teacher ID"
// @Param        level query string false "beginner|intermediate|advanced"
// @Param        status query string false "draft|published|archived"
// @Success      200 {object} CourseListEnvelope
// @Router       /api/v1/courses [get]
func (h *CourseHandler) ListCourses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	actorID := c.GetString(middleware.ContextUserID)
	admin := isPlatformAdmin(c) || rbac.HasPermission(domain.RoleCode(c.GetString(middleware.ContextRoleCode)), rbac.PermCoursesManage)

	filter := domain.CourseListFilter{
		Query:          c.Query("q"),
		Status:         domain.CourseStatus(c.Query("status")),
		CategorySlug:   c.Query("category"),
		TagSlug:        c.Query("tag"),
		OrganizationID: c.Query("organization_id"),
		TeacherID:      c.Query("teacher_id"),
		Level:          domain.CourseLevel(c.Query("level")),
		Page:           page,
		PerPage:        perPage,
		OwnerID:        actorID,
		Admin:          admin,
	}

	courses, total, filter, err := h.courses.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]CourseResponse, 0, len(courses))
	for i := range courses {
		out = append(out, toCourseResponse(&courses[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      filter.Page,
		PerPage:   filter.PerPage,
		Total:     total,
	})
}

// CreateCourse godoc
// @Summary      Create course
// @Tags         courses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createCourseRequest true "Course"
// @Success      201 {object} CourseEnvelope
// @Router       /api/v1/courses [post]
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	var req createCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	course, err := h.courses.Create(c.Request.Context(), c.GetString(middleware.ContextUserID), domain.CourseCreate{
		OrganizationID: req.OrganizationID,
		CategoryID:     req.CategoryID,
		Title:          req.Title,
		Slug:           req.Slug,
		Summary:        req.Summary,
		Description:    req.Description,
		CoverURL:       req.CoverURL,
		PriceCents:     req.PriceCents,
		Currency:       req.Currency,
		Level:          domain.CourseLevel(req.Level),
		Language:       req.Language,
		Tags:           req.Tags,
	}, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toCourseResponse(course))
}

// GetCourse godoc
// @Summary      Get course
// @Tags         courses
// @Produce      json
// @Param        id path string true "Course ID"
// @Success      200 {object} CourseEnvelope
// @Router       /api/v1/courses/{id} [get]
func (h *CourseHandler) GetCourse(c *gin.Context) {
	course, err := h.courses.Get(
		c.Request.Context(),
		c.Param("id"),
		c.GetString(middleware.ContextUserID),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCourseResponse(course))
}

// UpdateCourse godoc
// @Summary      Update course
// @Tags         courses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body updateCourseRequest true "Patch"
// @Success      200 {object} CourseEnvelope
// @Router       /api/v1/courses/{id} [patch]
func (h *CourseHandler) UpdateCourse(c *gin.Context) {
	var req updateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	patch := domain.CourseUpdate{
		CategoryID:  req.CategoryID,
		Title:       req.Title,
		Summary:     req.Summary,
		Description: req.Description,
		CoverURL:    req.CoverURL,
		PriceCents:  req.PriceCents,
		Currency:    req.Currency,
		Language:    req.Language,
	}
	if req.Level != nil {
		level := domain.CourseLevel(*req.Level)
		patch.Level = &level
	}
	course, err := h.courses.Update(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), patch, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCourseResponse(course))
}

// SetCourseTags godoc
// @Summary      Replace course tags
// @Tags         courses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body setTagsRequest true "Tags"
// @Success      200 {object} CourseEnvelope
// @Router       /api/v1/courses/{id}/tags [put]
func (h *CourseHandler) SetCourseTags(c *gin.Context) {
	var req setTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	course, err := h.courses.SetTags(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), req.Tags, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCourseResponse(course))
}

// PublishCourse godoc
// @Summary      Publish course
// @Tags         courses
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} CourseEnvelope
// @Router       /api/v1/courses/{id}/publish [post]
func (h *CourseHandler) PublishCourse(c *gin.Context) {
	course, err := h.courses.Publish(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCourseResponse(course))
}

// ArchiveCourse godoc
// @Summary      Archive course
// @Tags         courses
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} CourseEnvelope
// @Router       /api/v1/courses/{id}/archive [post]
func (h *CourseHandler) ArchiveCourse(c *gin.Context) {
	course, err := h.courses.Archive(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCourseResponse(course))
}

// DeleteCourse godoc
// @Summary      Soft-delete course
// @Tags         courses
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/courses/{id} [delete]
func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	if err := h.courses.Delete(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "course deleted"})
}

func toCourseResponse(c *domain.Course) CourseResponse {
	tags := c.TagSlugs
	if tags == nil {
		tags = []string{}
	}
	return CourseResponse{
		ID:                 c.ID,
		OrganizationID:     c.OrganizationID,
		TeacherID:          c.TeacherID,
		TeacherName:        c.TeacherName,
		CategoryID:         c.CategoryID,
		CategoryName:       c.CategoryName,
		Title:              c.Title,
		Slug:               c.Slug,
		Summary:            c.Summary,
		Description:        c.Description,
		Status:             string(c.Status),
		CoverURL:           c.CoverURL,
		PriceCents:         c.PriceCents,
		Currency:           c.Currency,
		Level:              string(c.Level),
		Language:           c.Language,
		Tags:               tags,
		EnrollmentCapacity: c.EnrollmentCapacity,
		EnrollmentOpen:     c.EnrollmentOpen,
		WaitlistEnabled:    c.WaitlistEnabled,
		PublishedAt:        c.PublishedAt,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

func toCategoryResponse(c domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:          c.ID,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		CreatedAt:   c.CreatedAt,
	}
}
