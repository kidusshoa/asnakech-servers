package domain

import "time"

// QuizStatus is draft/published for quizzes.
type QuizStatus string

const (
	QuizStatusDraft     QuizStatus = "draft"
	QuizStatusPublished QuizStatus = "published"
)

// QuestionType is mcq or short_answer.
type QuestionType string

const (
	QuestionTypeMCQ         QuestionType = "mcq"
	QuestionTypeShortAnswer QuestionType = "short_answer"
)

// AttemptStatus tracks a quiz attempt lifecycle.
type AttemptStatus string

const (
	AttemptStatusInProgress AttemptStatus = "in_progress"
	AttemptStatusSubmitted  AttemptStatus = "submitted"
	AttemptStatusGraded     AttemptStatus = "graded"
)

// AssignmentStatus is draft/published for assignments.
type AssignmentStatus string

const (
	AssignmentStatusDraft     AssignmentStatus = "draft"
	AssignmentStatusPublished AssignmentStatus = "published"
)

// SubmissionStatus tracks assignment submission lifecycle.
type SubmissionStatus string

const (
	SubmissionStatusDraft     SubmissionStatus = "draft"
	SubmissionStatusSubmitted SubmissionStatus = "submitted"
	SubmissionStatusGraded    SubmissionStatus = "graded"
	SubmissionStatusReturned  SubmissionStatus = "returned"
)

// QuizOption is one MCQ choice.
type QuizOption struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct,omitempty"`
}

// RubricCriterion is a simple assignment grading row.
type RubricCriterion struct {
	ID        string `json:"id"`
	Criterion string `json:"criterion"`
	MaxPoints int    `json:"max_points"`
}

// Quiz is a course assessment with questions.
type Quiz struct {
	ID               string
	CourseID         string
	Title            string
	Description      string
	Status           QuizStatus
	TimeLimitSeconds *int
	MaxAttempts      *int
	PassPercent      int
	ShuffleQuestions bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Questions        []QuizQuestion
}

// QuizQuestion is one quiz item.
type QuizQuestion struct {
	ID            string
	QuizID        string
	QuestionType  QuestionType
	Prompt        string
	Points        int
	Position      int
	Options       []QuizOption
	CorrectAnswer string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// QuizAttempt is one learner try at a quiz.
type QuizAttempt struct {
	ID            string
	QuizID        string
	UserID        string
	AttemptNumber int
	Status        AttemptStatus
	ScorePoints   int
	MaxPoints     int
	Percent       int
	Passed        bool
	StartedAt     time.Time
	SubmittedAt   *time.Time
	GradedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Answers       []QuizAttemptAnswer
	UserEmail     string
	UserFullName  string
}

// QuizAttemptAnswer stores a response for one question.
type QuizAttemptAnswer struct {
	ID                string
	AttemptID         string
	QuestionID        string
	SelectedOptionIDs []string
	TextAnswer        string
	IsCorrect         *bool
	PointsAwarded     int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Assignment is a course written/file task.
type Assignment struct {
	ID          string
	CourseID    string
	Title       string
	Description string
	Status      AssignmentStatus
	MaxScore    int
	DueAt       *time.Time
	AllowLate   bool
	Rubric      []RubricCriterion
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AssignmentSubmission is one learner submission.
type AssignmentSubmission struct {
	ID            string
	AssignmentID  string
	UserID        string
	Status        SubmissionStatus
	Body          string
	AttachmentURL string
	Score         *int
	Feedback      string
	RubricScores  map[string]int
	SubmittedAt   *time.Time
	GradedAt      *time.Time
	GradedBy      *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UserEmail     string
	UserFullName  string
}

// GradebookEntry aggregates learner assessment results for a course.
type GradebookEntry struct {
	UserID       string
	UserEmail    string
	UserFullName string
	Quizzes      []GradebookQuizScore
	Assignments  []GradebookAssignmentScore
}

// GradebookQuizScore is best graded attempt for a quiz.
type GradebookQuizScore struct {
	QuizID    string
	QuizTitle string
	Percent   *int
	Passed    *bool
	Attempts  int
}

// GradebookAssignmentScore is the graded submission score.
type GradebookAssignmentScore struct {
	AssignmentID    string
	AssignmentTitle string
	Score           *int
	MaxScore        int
	Status          string
}
