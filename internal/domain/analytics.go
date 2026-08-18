package domain

import "time"

// ReportRange filters time-bounded reports.
type ReportRange struct {
	From *time.Time
	To   *time.Time
}

// PlatformOverview is admin dashboard KPI snapshot.
type PlatformOverview struct {
	GeneratedAt time.Time

	UsersTotal       int64
	UsersByRole      map[string]int64
	CoursesTotal     int64
	CoursesByStatus  map[string]int64
	EnrollmentsTotal int64
	EnrollmentsByStatus map[string]int64
	OrganizationsTotal int64
	CertificatesTotal  int64

	OrdersPaid       int64
	RevenueCents     int64
	RevenueCurrency  string

	EnrollmentsLast7Days int64
	RevenueLast7DaysCents int64
	NewUsersLast7Days    int64
}

// CourseAnalytics summarizes a single course for teachers/admins.
type CourseAnalytics struct {
	CourseID    string
	CourseTitle string
	GeneratedAt time.Time

	EnrollmentsActive     int64
	EnrollmentsWaitlisted int64
	EnrollmentsCancelled  int64
	CompletionCount       int64
	CompletionRatePercent int
	AverageProgressPercent int

	OrdersPaid   int64
	RevenueCents int64
	Currency     string

	CertificatesIssued int64
	QuizzesPublished   int64
	AssignmentsPublished int64
}

// EnrollmentReportRow is one grouped enrollment metric.
type EnrollmentReportRow struct {
	Period     string
	CourseID   string
	CourseTitle string
	Status     string
	Count      int64
}

// RevenueReportRow is revenue grouped by period/course.
type RevenueReportRow struct {
	Period      string
	CourseID    string
	CourseTitle string
	Currency    string
	OrderCount  int64
	RevenueCents int64
}

// UserGrowthPoint is registrations over time.
type UserGrowthPoint struct {
	Period string
	Role   string
	Count  int64
}
