package domain

import "time"

// Certificate is a course completion credential with a public verification code.
type Certificate struct {
	ID               string
	CourseID         string
	UserID           string
	VerificationCode string
	LearnerName      string
	CourseTitle      string
	StorageKey       string
	PublicURL        string
	IssuedAt         time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	CourseSlug string
	UserEmail  string
}

// CertificateVerify is the public verification view.
type CertificateVerify struct {
	Valid            bool
	VerificationCode string
	LearnerName      string
	CourseTitle      string
	IssuedAt         time.Time
	RevokedAt        *time.Time
}

// Transcript is an exportable grade and progress summary.
type Transcript struct {
	UserID      string
	UserEmail   string
	UserFullName string
	GeneratedAt time.Time
	Courses     []TranscriptCourse
}

// TranscriptCourse is one course row on a transcript.
type TranscriptCourse struct {
	CourseID         string
	CourseTitle      string
	CourseSlug       string
	ProgressPercent  int
	CompletedAt      *time.Time
	Quizzes          []GradebookQuizScore
	Assignments      []GradebookAssignmentScore
	Certificate      *TranscriptCertificate
}

// TranscriptCertificate summarizes an issued credential on a transcript.
type TranscriptCertificate struct {
	ID               string
	VerificationCode string
	IssuedAt         time.Time
	Revoked          bool
}
