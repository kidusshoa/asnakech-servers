package handlers

import (
	"strconv"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/i18n"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type DiscoveryHandler struct {
	discovery *service.DiscoveryService
	parent    *service.ParentService
}

func NewDiscoveryHandler(discovery *service.DiscoveryService, parent *service.ParentService) *DiscoveryHandler {
	return &DiscoveryHandler{discovery: discovery, parent: parent}
}

type SearchResultsResponse struct {
	Query      string                    `json:"query"`
	Courses    []SearchCourseHitResponse `json:"courses"`
	Categories []CategoryResponse        `json:"categories"`
	Teachers   []SearchTeacherResponse   `json:"teachers"`
}

type SearchCourseHitResponse struct {
	CourseResponse
	Rank float32 `json:"rank"`
}

type SearchTeacherResponse struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type RecommendationResponse struct {
	CourseResponse
	Reason string `json:"reason"`
	Score  int    `json:"score"`
}

type SearchEnvelope struct {
	Success bool                  `json:"success" example:"true"`
	Data    SearchResultsResponse `json:"data"`
}

type RecommendationListEnvelope struct {
	Success bool                     `json:"success" example:"true"`
	Data    []RecommendationResponse `json:"data"`
}

type FeatureFlagsEnvelope struct {
	Success bool            `json:"success" example:"true"`
	Data    map[string]bool `json:"data"`
}

type LocaleInfoResponse struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
	RTL        bool   `json:"rtl"`
}

type LocalesEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []LocaleInfoResponse `json:"data"`
}

type MessagesEnvelope struct {
	Success bool              `json:"success" example:"true"`
	Data    map[string]string `json:"data"`
}

type ParentLinkResponse struct {
	ID              string `json:"id"`
	StudentID       string `json:"student_id"`
	StudentEmail    string `json:"student_email"`
	StudentFullName string `json:"student_full_name"`
	Status          string `json:"status"`
}

type linkChildRequest struct {
	StudentEmail string `json:"student_email" binding:"required"`
}

type ParentLinkEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    ParentLinkResponse `json:"data"`
}

type ParentLinkListEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []ParentLinkResponse `json:"data"`
}

// Search godoc
// @Summary Unified catalog search
// @Description Full-text search across published courses, categories, and teachers
// @Tags discovery
// @Produce json
// @Param q query string true "Search query"
// @Param type query string false "all|courses|categories|teachers"
// @Param language query string false "Course language filter"
// @Param level query string false "Course level filter"
// @Param page query int false "Page (courses only)"
// @Param per_page query int false "Per page (courses only)"
// @Success 200 {object} SearchEnvelope
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/search [get]
func (h *DiscoveryHandler) Search(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	results, err := h.discovery.Search(c.Request.Context(), domain.SearchFilter{
		Query:    c.Query("q"),
		Type:     domain.SearchType(c.Query("type")),
		Language: c.Query("language"),
		Level:    domain.CourseLevel(c.Query("level")),
		Page:     page,
		PerPage:  perPage,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toSearchResultsResponse(results))
}

// Recommendations godoc
// @Summary Personalized course recommendations
// @Tags discovery
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Max results (default 10)"
// @Success 200 {object} RecommendationListEnvelope
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/me/recommendations [get]
func (h *DiscoveryHandler) Recommendations(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	locale := c.GetString(i18n.ContextLocale)
	items, err := h.discovery.Recommendations(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		locale,
		limit,
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]RecommendationResponse, len(items))
	for i := range items {
		out[i] = RecommendationResponse{
			CourseResponse: toCourseResponse(&items[i].Course),
			Reason:         items[i].Reason,
			Score:          items[i].Score,
		}
	}
	response.OK(c, out)
}

// FeatureFlags godoc
// @Summary Platform feature flags
// @Tags discovery
// @Produce json
// @Success 200 {object} FeatureFlagsEnvelope
// @Router /api/v1/features [get]
func (h *DiscoveryHandler) FeatureFlags(c *gin.Context) {
	response.OK(c, h.discovery.FeatureFlags())
}

// Locales godoc
// @Summary Supported UI locales
// @Tags discovery
// @Produce json
// @Success 200 {object} LocalesEnvelope
// @Router /api/v1/locales [get]
func (h *DiscoveryHandler) Locales(c *gin.Context) {
	locales := h.discovery.Locales()
	out := make([]LocaleInfoResponse, len(locales))
	for i, l := range locales {
		out[i] = LocaleInfoResponse{
			Code: l.Code, Name: l.Name, NativeName: l.NativeName, RTL: l.RTL,
		}
	}
	response.OK(c, out)
}

// Messages godoc
// @Summary Localized UI messages
// @Tags discovery
// @Produce json
// @Param lang query string false "Locale code (en, am)"
// @Success 200 {object} MessagesEnvelope
// @Router /api/v1/i18n/messages [get]
func (h *DiscoveryHandler) Messages(c *gin.Context) {
	locale := c.Query("lang")
	if locale == "" {
		locale = c.GetString(i18n.ContextLocale)
	}
	response.OK(c, h.discovery.Messages(locale))
}

// LinkChild godoc
// @Summary Link a student to parent account
// @Tags parents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body linkChildRequest true "Student email"
// @Success 201 {object} ParentLinkEnvelope
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/me/children/link [post]
func (h *DiscoveryHandler) LinkChild(c *gin.Context) {
	var req linkChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	link, err := h.parent.LinkChild(c.Request.Context(), c.GetString(middleware.ContextUserID), req.StudentEmail)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toParentLinkResponse(link))
}

// ListChildren godoc
// @Summary List linked students
// @Tags parents
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ParentLinkListEnvelope
// @Router /api/v1/me/children [get]
func (h *DiscoveryHandler) ListChildren(c *gin.Context) {
	links, err := h.parent.ListChildren(c.Request.Context(), c.GetString(middleware.ContextUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]ParentLinkResponse, len(links))
	for i := range links {
		out[i] = toParentLinkResponse(&links[i])
	}
	response.OK(c, out)
}

// UnlinkChild godoc
// @Summary Revoke parent-student link
// @Tags parents
// @Produce json
// @Security BearerAuth
// @Param studentId path string true "Student user ID"
// @Success 204 "No Content"
// @Router /api/v1/me/children/{studentId} [delete]
func (h *DiscoveryHandler) UnlinkChild(c *gin.Context) {
	if err := h.parent.UnlinkChild(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("studentId")); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(204)
}

func toSearchResultsResponse(r *domain.SearchResults) SearchResultsResponse {
	if r == nil {
		return SearchResultsResponse{}
	}
	out := SearchResultsResponse{
		Query:      r.Query,
		Categories: make([]CategoryResponse, len(r.Categories)),
		Teachers:   make([]SearchTeacherResponse, len(r.Teachers)),
		Courses:    make([]SearchCourseHitResponse, len(r.Courses)),
	}
	for i, c := range r.Categories {
		out.Categories[i] = toCategoryResponse(c)
	}
	for i, t := range r.Teachers {
		out.Teachers[i] = SearchTeacherResponse{ID: t.ID, FullName: t.FullName, Email: t.Email}
	}
	for i, hit := range r.Courses {
		out.Courses[i] = SearchCourseHitResponse{
			CourseResponse: toCourseResponse(&hit.Course),
			Rank:           hit.Rank,
		}
	}
	return out
}

func toParentLinkResponse(l *domain.ParentStudentLink) ParentLinkResponse {
	if l == nil {
		return ParentLinkResponse{}
	}
	return ParentLinkResponse{
		ID:              l.ID,
		StudentID:       l.StudentUserID,
		StudentEmail:    l.StudentEmail,
		StudentFullName: l.StudentFullName,
		Status:          l.Status,
	}
}
