package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type AnalyticsRepository interface {
	PlatformOverview(ctx context.Context) (*domain.PlatformOverview, error)
	CourseAnalytics(ctx context.Context, courseID string) (*domain.CourseAnalytics, error)
	EnrollmentReport(ctx context.Context, filter domain.ReportRange) ([]domain.EnrollmentReportRow, error)
	RevenueReport(ctx context.Context, filter domain.ReportRange) ([]domain.RevenueReportRow, error)
	UserGrowthReport(ctx context.Context, filter domain.ReportRange) ([]domain.UserGrowthPoint, error)
}
