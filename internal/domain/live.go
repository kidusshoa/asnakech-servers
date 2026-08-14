package domain

import "time"

// LiveSessionStatus is the lifecycle of a scheduled class session.
type LiveSessionStatus string

const (
	LiveSessionStatusDraft     LiveSessionStatus = "draft"
	LiveSessionStatusScheduled LiveSessionStatus = "scheduled"
	LiveSessionStatusCompleted LiveSessionStatus = "completed"
	LiveSessionStatusCancelled LiveSessionStatus = "cancelled"
)

// LiveProvider is the video-conference integration adapter.
type LiveProvider string

const (
	LiveProviderCustom     LiveProvider = "custom"
	LiveProviderJitsi      LiveProvider = "jitsi"
	LiveProviderZoom       LiveProvider = "zoom"
	LiveProviderGoogleMeet LiveProvider = "google_meet"
)

// AttendanceStatus records participation for a session.
type AttendanceStatus string

const (
	AttendanceStatusRegistered AttendanceStatus = "registered"
	AttendanceStatusPresent    AttendanceStatus = "present"
	AttendanceStatusAbsent     AttendanceStatus = "absent"
	AttendanceStatusLate       AttendanceStatus = "late"
	AttendanceStatusExcused    AttendanceStatus = "excused"
)

// LiveSession is a scheduled live class tied to a course.
type LiveSession struct {
	ID               string
	CourseID         string
	LessonID         *string
	Title            string
	Description      string
	Status           LiveSessionStatus
	StartsAt         time.Time
	EndsAt           time.Time
	Timezone         string
	Provider         LiveProvider
	JoinURL          string
	HostURL          string
	ExternalID       string
	ProviderMetadata map[string]string
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Optional joined fields
	CourseTitle string
	CourseSlug  string
}

// LiveSessionCreate is input for creating a session.
type LiveSessionCreate struct {
	LessonID    *string
	Title       string
	Description string
	StartsAt    time.Time
	EndsAt      time.Time
	Timezone    string
	Provider    LiveProvider
	JoinURL     string
	HostURL     string
}

// LiveSessionUpdate patches mutable session fields.
type LiveSessionUpdate struct {
	LessonID    *string
	Title       *string
	Description *string
	StartsAt    *time.Time
	EndsAt      *time.Time
	Timezone    *string
	Provider    *LiveProvider
	JoinURL     *string
	HostURL     *string
}

// SessionAttendance links a user to a live session.
type SessionAttendance struct {
	ID        string
	SessionID string
	UserID    string
	Status    AttendanceStatus
	JoinedAt  *time.Time
	LeftAt    *time.Time
	MarkedBy  *string
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time

	UserEmail    string
	UserFullName string
}

// CalendarFilter selects sessions for a user's feed.
type CalendarFilter struct {
	From    time.Time
	To      time.Time
	UserID  string
	Admin   bool
	Page    int
	PerPage int
}

// JoinInfo is returned when a participant opens a session.
type JoinInfo struct {
	SessionID string
	JoinURL   string
	HostURL   string
	StartsAt  time.Time
	EndsAt    time.Time
	Status    LiveSessionStatus
	IsHost    bool
}
