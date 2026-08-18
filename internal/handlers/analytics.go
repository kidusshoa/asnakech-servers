package handlers

import (
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analytics *service.AnalyticsService
}

func NewAnalyticsHandler(analytics *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics}
}

type PlatformOverviewResponse struct {
	GeneratedAt string            `json:"generated_at"`
	Users       usersOverview     `json:"users"`
	Courses     coursesOverview   `json:"courses"`
	Enrollments enrollOverview    `json:"enrollments"`
	Organizations int64           `json:"organizations_total"`
	Certificates  int64           `json:"certificates_total"`
	Revenue       revenueOverview `json:"revenue"`
	Trends        trendsOverview  `json:"trends"`
}

type usersOverview struct {
	Total   int64            `json:"total"`
	ByRole  map[string]int64 `json:"by_role"`
}

type coursesOverview struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
}

type enrollOverview struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
}

type revenueOverview struct {
	OrdersPaid  int64  `json:"orders_paid"`
	TotalCents  int64  `json:"total_cents"`
	Currency    string `json:"currency"`
}

type trendsOverview struct {
	EnrollmentsLast7Days int64 `json:"enrollments_last_7_days"`
	RevenueLast7DaysCents int64 `json:"revenue_last_7_days_cents"`
	NewUsersLast7Days    int64 `json:"new_users_last_7_days"`
}

type CourseAnalyticsResponse struct {
	CourseID               string `json:"course_id"`
	CourseTitle            string `json:"course_title"`
	GeneratedAt            string `json:"generated_at"`
	EnrollmentsActive      int64  `json:"enrollments_active"`
	EnrollmentsWaitlisted  int64  `json:"enrollments_waitlisted"`
	EnrollmentsCancelled   int64  `json:"enrollments_cancelled"`
	CompletionCount        int64  `json:"completion_count"`
	CompletionRatePercent  int    `json:"completion_rate_percent"`
	AverageProgressPercent int    `json:"average_progress_percent"`
	OrdersPaid             int64  `json:"orders_paid"`
	RevenueCents           int64  `json:"revenue_cents"`
	Currency               string `json:"currency"`
	CertificatesIssued     int64  `json:"certificates_issued"`
	QuizzesPublished       int64  `json:"quizzes_published"`
	AssignmentsPublished   int64  `json:"assignments_published"`
}

type EnrollmentReportRowResponse struct {
	Period      string `json:"period"`
	CourseID    string `json:"course_id"`
	CourseTitle string `json:"course_title"`
	Status      string `json:"status"`
	Count       int64  `json:"count"`
}

type RevenueReportRowResponse struct {
	Period       string `json:"period"`
	CourseID     string `json:"course_id"`
	CourseTitle  string `json:"course_title"`
	Currency     string `json:"currency"`
	OrderCount   int64  `json:"order_count"`
	RevenueCents int64  `json:"revenue_cents"`
}

type UserGrowthPointResponse struct {
	Period string `json:"period"`
	Role   string `json:"role"`
	Count  int64  `json:"count"`
}

type PlatformOverviewEnvelope struct {
	Success bool                     `json:"success" example:"true"`
	Data    PlatformOverviewResponse `json:"data"`
}

type CourseAnalyticsEnvelope struct {
	Success bool                    `json:"success" example:"true"`
	Data    CourseAnalyticsResponse `json:"data"`
}

type EnrollmentReportEnvelope struct {
	Success bool                        `json:"success" example:"true"`
	Data    []EnrollmentReportRowResponse `json:"data"`
}

type RevenueReportEnvelope struct {
	Success bool                     `json:"success" example:"true"`
	Data    []RevenueReportRowResponse `json:"data"`
}

type UserGrowthReportEnvelope struct {
	Success bool                      `json:"success" example:"true"`
	Data    []UserGrowthPointResponse `json:"data"`
}

// AdminOverview godoc
// @Summary Platform overview (admin)
// @Description KPI snapshot: users, courses, enrollments, revenue, 7-day trends
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PlatformOverviewEnvelope
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/admin/overview [get]
func (h *AnalyticsHandler) AdminOverview(c *gin.Context) {
	overview, err := h.analytics.PlatformOverview(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toPlatformOverviewResponse(overview))
}

// EnrollmentReport godoc
// @Summary Enrollment report (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param from query string false "Start date (YYYY-MM-DD or RFC3339)"
// @Param to query string false "End date (YYYY-MM-DD or RFC3339)"
// @Success 200 {object} EnrollmentReportEnvelope
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/admin/reports/enrollments [get]
func (h *AnalyticsHandler) EnrollmentReport(c *gin.Context) {
	filter, err := reportFilterFromQuery(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	rows, err := h.analytics.EnrollmentReport(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toEnrollmentReportRows(rows))
}

// RevenueReport godoc
// @Summary Revenue report (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param from query string false "Start date"
// @Param to query string false "End date"
// @Success 200 {object} RevenueReportEnvelope
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/admin/reports/revenue [get]
func (h *AnalyticsHandler) RevenueReport(c *gin.Context) {
	filter, err := reportFilterFromQuery(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	rows, err := h.analytics.RevenueReport(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toRevenueReportRows(rows))
}

// UserGrowthReport godoc
// @Summary User growth report (admin)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param from query string false "Start date"
// @Param to query string false "End date"
// @Success 200 {object} UserGrowthReportEnvelope
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/admin/reports/users [get]
func (h *AnalyticsHandler) UserGrowthReport(c *gin.Context) {
	filter, err := reportFilterFromQuery(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	rows, err := h.analytics.UserGrowthReport(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserGrowthPoints(rows))
}

// CourseAnalytics godoc
// @Summary Course analytics (teacher)
// @Tags analytics
// @Produce json
// @Security BearerAuth
// @Param id path string true "Course ID"
// @Success 200 {object} CourseAnalyticsEnvelope
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/courses/{id}/analytics [get]
func (h *AnalyticsHandler) CourseAnalytics(c *gin.Context) {
	stats, err := h.analytics.CourseAnalytics(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCourseAnalyticsResponse(stats))
}

func reportFilterFromQuery(c *gin.Context) (domain.ReportRange, error) {
	return service.ParseReportRange(c.Query("from"), c.Query("to"))
}

func toPlatformOverviewResponse(o *domain.PlatformOverview) PlatformOverviewResponse {
	if o == nil {
		return PlatformOverviewResponse{}
	}
	return PlatformOverviewResponse{
		GeneratedAt: o.GeneratedAt.Format(time.RFC3339),
		Users: usersOverview{
			Total:  o.UsersTotal,
			ByRole: o.UsersByRole,
		},
		Courses: coursesOverview{
			Total:    o.CoursesTotal,
			ByStatus: o.CoursesByStatus,
		},
		Enrollments: enrollOverview{
			Total:    o.EnrollmentsTotal,
			ByStatus: o.EnrollmentsByStatus,
		},
		Organizations: o.OrganizationsTotal,
		Certificates:  o.CertificatesTotal,
		Revenue: revenueOverview{
			OrdersPaid: o.OrdersPaid,
			TotalCents: o.RevenueCents,
			Currency:   o.RevenueCurrency,
		},
		Trends: trendsOverview{
			EnrollmentsLast7Days:  o.EnrollmentsLast7Days,
			RevenueLast7DaysCents: o.RevenueLast7DaysCents,
			NewUsersLast7Days:     o.NewUsersLast7Days,
		},
	}
}

func toCourseAnalyticsResponse(a *domain.CourseAnalytics) CourseAnalyticsResponse {
	if a == nil {
		return CourseAnalyticsResponse{}
	}
	return CourseAnalyticsResponse{
		CourseID:               a.CourseID,
		CourseTitle:            a.CourseTitle,
		GeneratedAt:            a.GeneratedAt.Format(time.RFC3339),
		EnrollmentsActive:      a.EnrollmentsActive,
		EnrollmentsWaitlisted:  a.EnrollmentsWaitlisted,
		EnrollmentsCancelled:   a.EnrollmentsCancelled,
		CompletionCount:        a.CompletionCount,
		CompletionRatePercent:  a.CompletionRatePercent,
		AverageProgressPercent: a.AverageProgressPercent,
		OrdersPaid:             a.OrdersPaid,
		RevenueCents:           a.RevenueCents,
		Currency:               a.Currency,
		CertificatesIssued:     a.CertificatesIssued,
		QuizzesPublished:       a.QuizzesPublished,
		AssignmentsPublished:   a.AssignmentsPublished,
	}
}

func toEnrollmentReportRows(rows []domain.EnrollmentReportRow) []EnrollmentReportRowResponse {
	out := make([]EnrollmentReportRowResponse, len(rows))
	for i, r := range rows {
		out[i] = EnrollmentReportRowResponse{
			Period:      r.Period,
			CourseID:    r.CourseID,
			CourseTitle: r.CourseTitle,
			Status:      r.Status,
			Count:       r.Count,
		}
	}
	return out
}

func toRevenueReportRows(rows []domain.RevenueReportRow) []RevenueReportRowResponse {
	out := make([]RevenueReportRowResponse, len(rows))
	for i, r := range rows {
		out[i] = RevenueReportRowResponse{
			Period:       r.Period,
			CourseID:     r.CourseID,
			CourseTitle:  r.CourseTitle,
			Currency:     r.Currency,
			OrderCount:   r.OrderCount,
			RevenueCents: r.RevenueCents,
		}
	}
	return out
}

func toUserGrowthPoints(rows []domain.UserGrowthPoint) []UserGrowthPointResponse {
	out := make([]UserGrowthPointResponse, len(rows))
	for i, r := range rows {
		out[i] = UserGrowthPointResponse{
			Period: r.Period,
			Role:   r.Role,
			Count:  r.Count,
		}
	}
	return out
}
