package domain

import "time"

// SearchType limits unified search results.
type SearchType string

const (
	SearchTypeAll        SearchType = "all"
	SearchTypeCourses    SearchType = "courses"
	SearchTypeCategories SearchType = "categories"
	SearchTypeTeachers   SearchType = "teachers"
)

// SearchFilter controls discovery queries.
type SearchFilter struct {
	Query    string
	Type     SearchType
	Language string
	Level    CourseLevel
	Page     int
	PerPage  int
}

// SearchCourseHit is a ranked course match.
type SearchCourseHit struct {
	Course
	Rank float32
}

// SearchTeacherHit is a teacher matching a query.
type SearchTeacherHit struct {
	ID       string
	FullName string
	Email    string
}

// SearchResults groups unified search output.
type SearchResults struct {
	Query      string
	Courses    []SearchCourseHit
	Categories []Category
	Teachers   []SearchTeacherHit
}

// CourseRecommendation is a suggested course for a learner.
type CourseRecommendation struct {
	Course
	Reason string
	Score  int
}

// ParentStudentLink connects a guardian to a learner.
type ParentStudentLink struct {
	ID            string
	ParentUserID  string
	StudentUserID string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	StudentEmail    string
	StudentFullName string
}

// LocaleInfo describes a supported UI language.
type LocaleInfo struct {
	Code       string
	Name       string
	NativeName string
	RTL        bool
}
