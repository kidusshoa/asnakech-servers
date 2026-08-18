package service

import (
	"context"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type AnalyticsService struct {
	courses   repository.CourseRepository
	analytics repository.AnalyticsRepository
}

func NewAnalyticsService(
	courses repository.CourseRepository,
	analytics repository.AnalyticsRepository,
) *AnalyticsService {
	return &AnalyticsService{courses: courses, analytics: analytics}
}

func (s *AnalyticsService) PlatformOverview(ctx context.Context) (*domain.PlatformOverview, error) {
	return s.analytics.PlatformOverview(ctx)
}

func (s *AnalyticsService) CourseAnalytics(ctx context.Context, actorID, courseID string, platformAdmin bool) (*domain.CourseAnalytics, error) {
	if err := s.requireCourseView(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.analytics.CourseAnalytics(ctx, courseID)
}

func (s *AnalyticsService) EnrollmentReport(ctx context.Context, filter domain.ReportRange) ([]domain.EnrollmentReportRow, error) {
	return s.analytics.EnrollmentReport(ctx, filter)
}

func (s *AnalyticsService) RevenueReport(ctx context.Context, filter domain.ReportRange) ([]domain.RevenueReportRow, error) {
	return s.analytics.RevenueReport(ctx, filter)
}

func (s *AnalyticsService) UserGrowthReport(ctx context.Context, filter domain.ReportRange) ([]domain.UserGrowthPoint, error) {
	return s.analytics.UserGrowthReport(ctx, filter)
}

func (s *AnalyticsService) requireCourseView(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	return apperr.Forbidden("only the course teacher or an admin can view analytics")
}

func ParseReportRange(fromRaw, toRaw string) (domain.ReportRange, error) {
	var filter domain.ReportRange
	if fromRaw != "" {
		t, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			if t2, err2 := time.Parse("2006-01-02", fromRaw); err2 == nil {
				filter.From = &t2
			} else {
				return filter, apperr.Validation("from must be YYYY-MM-DD or RFC3339")
			}
		} else {
			filter.From = &t
		}
	}
	if toRaw != "" {
		t, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			if t2, err2 := time.Parse("2006-01-02", toRaw); err2 == nil {
				// end of day inclusive
				end := t2.Add(24 * time.Hour)
				filter.To = &end
			} else {
				return filter, apperr.Validation("to must be YYYY-MM-DD or RFC3339")
			}
		} else {
			filter.To = &t
		}
	}
	return filter, nil
}
