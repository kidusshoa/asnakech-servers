package domain

import "time"

// CourseStatus is the publication lifecycle of a course.
type CourseStatus string

const (
	CourseStatusDraft     CourseStatus = "draft"
	CourseStatusPublished CourseStatus = "published"
	CourseStatusArchived  CourseStatus = "archived"
)

// CourseLevel describes difficulty.
type CourseLevel string

const (
	CourseLevelBeginner     CourseLevel = "beginner"
	CourseLevelIntermediate CourseLevel = "intermediate"
	CourseLevelAdvanced     CourseLevel = "advanced"
)

// Category groups courses in the catalog.
type Category struct {
	ID          string
	Name        string
	Slug        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Tag is a free-form course label.
type Tag struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
}

// Course is a catalog entry authored by a teacher.
type Course struct {
	ID             string
	OrganizationID *string
	TeacherID      string
	CategoryID     *string
	Title          string
	Slug           string
	Summary        string
	Description    string
	Status         CourseStatus
	CoverURL       string
	PriceCents     int
	Currency       string
	Level          CourseLevel
	Language       string
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Optional joined fields
	CategoryName string
	TeacherName  string
	TagSlugs     []string
}

// CourseCreate is input for creating a course.
type CourseCreate struct {
	OrganizationID *string
	CategoryID     *string
	Title          string
	Slug           string
	Summary        string
	Description    string
	CoverURL       string
	PriceCents     int
	Currency       string
	Level          CourseLevel
	Language       string
	Tags           []string
}

// CourseUpdate patches mutable course fields.
type CourseUpdate struct {
	CategoryID  *string
	Title       *string
	Summary     *string
	Description *string
	CoverURL    *string
	PriceCents  *int
	Currency    *string
	Level       *CourseLevel
	Language    *string
}

// CourseListFilter controls catalog listing.
type CourseListFilter struct {
	Query          string
	Status         CourseStatus // empty = published for public; teachers may pass draft
	CategorySlug   string
	TagSlug        string
	OrganizationID string
	TeacherID      string
	Level          CourseLevel
	Page           int
	PerPage        int
	// IncludeNonPublished allows teacher/admin private visibility when set with OwnerID/Admin.
	OwnerID string
	Admin   bool
}
