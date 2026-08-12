package domain

import "time"

// LessonProgressStatus tracks a learner's work on one lesson.
type LessonProgressStatus string

const (
	LessonProgressInProgress LessonProgressStatus = "in_progress"
	LessonProgressCompleted  LessonProgressStatus = "completed"
)

// LessonProgress is per-user progress on a single lesson.
type LessonProgress struct {
	ID           string
	UserID       string
	CourseID     string
	LessonID     string
	Status       LessonProgressStatus
	Percent      int
	LastPosition string
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time

	LessonTitle string
	LessonSlug  string
}

// CourseProgress aggregates lesson completion for a course enrollment.
type CourseProgress struct {
	ID               string
	UserID           string
	CourseID         string
	EnrollmentID     *string
	Percent          int
	CompletedLessons int
	TotalLessons     int
	LastLessonID     *string
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	CourseTitle string
	CourseSlug  string
}

// LessonProgressUpsert is an idempotent progress write.
type LessonProgressUpsert struct {
	Percent      *int
	LastPosition *string
	Completed    *bool
}

// CourseProgressSummary is the student dashboard row.
type CourseProgressSummary struct {
	CourseProgress
	EnrollmentStatus EnrollmentStatus
}
