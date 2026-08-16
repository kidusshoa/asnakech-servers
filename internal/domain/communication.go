package domain

import "time"

type AnnouncementStatus string

const (
	AnnouncementStatusDraft     AnnouncementStatus = "draft"
	AnnouncementStatusPublished AnnouncementStatus = "published"
)

type ThreadStatus string

const (
	ThreadStatusOpen   ThreadStatus = "open"
	ThreadStatusLocked ThreadStatus = "locked"
)

type NotificationChannel string

const (
	NotificationChannelInApp NotificationChannel = "in_app"
	NotificationChannelEmail NotificationChannel = "email"
)

type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

type Announcement struct {
	ID          string
	CourseID    string
	AuthorID    string
	Title       string
	Body        string
	Status      AnnouncementStatus
	Pinned      bool
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time

	AuthorName string
}

type DiscussionThread struct {
	ID        string
	CourseID  string
	AuthorID  string
	Title     string
	Status    ThreadStatus
	CreatedAt time.Time
	UpdatedAt time.Time

	AuthorName string
	PostCount  int
}

type DiscussionPost struct {
	ID        string
	ThreadID  string
	AuthorID  string
	ParentID  *string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time

	AuthorName string
}

type DMConversation struct {
	ID        string
	UserAID   string
	UserBID   string
	CreatedAt time.Time
	UpdatedAt time.Time

	OtherUserID   string
	OtherUserName string
	LastMessage   string
	UnreadCount   int
}

type DMMessage struct {
	ID             string
	ConversationID string
	SenderID       string
	Body           string
	ReadAt         *time.Time
	CreatedAt      time.Time
}

type Notification struct {
	ID        string
	UserID    string
	Channel   NotificationChannel
	EventType string
	Subject   string
	Body      string
	Payload   map[string]string
	Status    NotificationStatus
	ReadAt    *time.Time
	CreatedAt time.Time
	SentAt    *time.Time
}

type NotificationListFilter struct {
	UnreadOnly bool
	Page       int
	PerPage    int
}
