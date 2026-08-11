package domain

import "time"

// LessonStatus is draft/published for individual lessons.
type LessonStatus string

const (
	LessonStatusDraft     LessonStatus = "draft"
	LessonStatusPublished LessonStatus = "published"
)

// ContentBlockType identifies a lesson content unit.
type ContentBlockType string

const (
	ContentBlockText    ContentBlockType = "text"
	ContentBlockVideo   ContentBlockType = "video"
	ContentBlockFile    ContentBlockType = "file"
	ContentBlockQuizRef ContentBlockType = "quiz_ref"
)

// CourseModule groups lessons inside a course.
type CourseModule struct {
	ID        string
	CourseID  string
	Title     string
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
	Lessons   []Lesson
}

// Lesson is a unit of learning within a module.
type Lesson struct {
	ID                   string
	ModuleID             string
	Title                string
	Slug                 string
	Summary              string
	Status               LessonStatus
	Position             int
	PrerequisiteLessonID *string
	EstimatedMinutes     int
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Blocks               []ContentBlock
}

// ContentBlock is an ordered piece of lesson content.
type ContentBlock struct {
	ID        string
	LessonID  string
	BlockType ContentBlockType
	Title     string
	Body      string
	MediaURL  string
	QuizRefID *string
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CurriculumTree is the nested authoring/view payload for a course.
type CurriculumTree struct {
	CourseID string
	Modules  []CourseModule
}
