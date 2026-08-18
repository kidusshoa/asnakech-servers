package domain

import "time"

// EnrollmentStatus is the learner's seat state on a course.
type EnrollmentStatus string

const (
	EnrollmentStatusActive     EnrollmentStatus = "active"
	EnrollmentStatusWaitlisted EnrollmentStatus = "waitlisted"
	EnrollmentStatusCancelled  EnrollmentStatus = "cancelled"
)

// EnrollmentSource describes how the seat was obtained.
type EnrollmentSource string

const (
	EnrollmentSourceSelf       EnrollmentSource = "self"
	EnrollmentSourceInviteCode EnrollmentSource = "invite_code"
	EnrollmentSourceTeacher    EnrollmentSource = "teacher"
	EnrollmentSourcePayment    EnrollmentSource = "payment"
)

// EnrollmentEventType is appended for later notification workers.
type EnrollmentEventType string

const (
	EnrollmentEventEnrolled   EnrollmentEventType = "enrolled"
	EnrollmentEventWaitlisted EnrollmentEventType = "waitlisted"
	EnrollmentEventActivated  EnrollmentEventType = "activated"
	EnrollmentEventCancelled  EnrollmentEventType = "cancelled"
)

// Enrollment links a user to a course seat.
type Enrollment struct {
	ID           string
	CourseID     string
	UserID       string
	Status       EnrollmentStatus
	Source       EnrollmentSource
	InviteCodeID *string
	EnrolledAt   *time.Time
	WaitlistedAt *time.Time
	CancelledAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Optional joined fields
	UserEmail    string
	UserFullName string
	CourseTitle  string
	CourseSlug   string
}

// EnrollmentInviteCode is a shareable enrollment key for a course.
type EnrollmentInviteCode struct {
	ID        string
	CourseID  string
	Code      string
	MaxUses   *int
	UsesCount int
	ExpiresAt *time.Time
	CreatedBy string
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// EnrollmentEvent is an append-only lifecycle record.
type EnrollmentEvent struct {
	ID           string
	EnrollmentID string
	CourseID     string
	UserID       string
	EventType    EnrollmentEventType
	CreatedAt    time.Time
}

// CourseEnrollmentSettings controls capacity and open/invite-only enrollment.
type CourseEnrollmentSettings struct {
	Capacity        *int
	EnrollmentOpen  bool
	WaitlistEnabled bool
}

// EnrollmentListFilter paginates enrollment listings.
type EnrollmentListFilter struct {
	Page    int
	PerPage int
	Status  EnrollmentStatus // empty = all
}

// CourseAccess summarizes whether the actor can consume content.
type CourseAccess struct {
	CourseID         string
	CanAccessContent bool
	IsTeacher        bool
	IsPlatformAdmin  bool
	Enrollment       *Enrollment
}
