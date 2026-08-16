package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type AnnouncementRepository interface {
	Create(ctx context.Context, a *domain.Announcement) error
	GetByID(ctx context.Context, id string) (*domain.Announcement, error)
	ListByCourse(ctx context.Context, courseID string, includeDraft bool) ([]domain.Announcement, error)
	Update(ctx context.Context, a *domain.Announcement) (*domain.Announcement, error)
	SetStatus(ctx context.Context, id string, status domain.AnnouncementStatus, publishedAt *domain.Announcement) error
	Delete(ctx context.Context, id string) error
}

type DiscussionThreadRepository interface {
	Create(ctx context.Context, t *domain.DiscussionThread) error
	GetByID(ctx context.Context, id string) (*domain.DiscussionThread, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.DiscussionThread, error)
	Update(ctx context.Context, t *domain.DiscussionThread) (*domain.DiscussionThread, error)
	SetStatus(ctx context.Context, id string, status domain.ThreadStatus) (*domain.DiscussionThread, error)
}

type DiscussionPostRepository interface {
	Create(ctx context.Context, p *domain.DiscussionPost) error
	GetByID(ctx context.Context, id string) (*domain.DiscussionPost, error)
	ListByThread(ctx context.Context, threadID string) ([]domain.DiscussionPost, error)
	Update(ctx context.Context, p *domain.DiscussionPost) (*domain.DiscussionPost, error)
	Delete(ctx context.Context, id string) error
	ListParticipantIDs(ctx context.Context, threadID string) ([]string, error)
}

type DMConversationRepository interface {
	GetOrCreate(ctx context.Context, userAID, userBID string) (*domain.DMConversation, error)
	GetByID(ctx context.Context, id string) (*domain.DMConversation, error)
	ListForUser(ctx context.Context, userID string) ([]domain.DMConversation, error)
}

type DMMessageRepository interface {
	Create(ctx context.Context, m *domain.DMMessage) error
	ListByConversation(ctx context.Context, conversationID string, limit, offset int) ([]domain.DMMessage, int, error)
	MarkRead(ctx context.Context, conversationID, readerID string) error
}

type NotificationRepository interface {
	Enqueue(ctx context.Context, n *domain.Notification) error
	ListForUser(ctx context.Context, userID string, filter domain.NotificationListFilter) ([]domain.Notification, int, error)
	GetByID(ctx context.Context, id, userID string) (*domain.Notification, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
}
