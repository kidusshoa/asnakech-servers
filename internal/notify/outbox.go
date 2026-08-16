package notify

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

// OutboxWriter enqueues notifications for async delivery (email worker later).
type OutboxWriter struct {
	notifications repository.NotificationRepository
}

func NewOutboxWriter(notifications repository.NotificationRepository) *OutboxWriter {
	return &OutboxWriter{notifications: notifications}
}

func (w *OutboxWriter) EnqueueInApp(ctx context.Context, userID, eventType, subject, body string, payload map[string]string) error {
	if payload == nil {
		payload = map[string]string{}
	}
	n := &domain.Notification{
		UserID:    userID,
		Channel:   domain.NotificationChannelInApp,
		EventType: eventType,
		Subject:   subject,
		Body:      body,
		Payload:   payload,
		Status:    domain.NotificationStatusSent,
	}
	return w.notifications.Enqueue(ctx, n)
}

func (w *OutboxWriter) EnqueueEmail(ctx context.Context, userID, eventType, subject, body string, payload map[string]string) error {
	if payload == nil {
		payload = map[string]string{}
	}
	n := &domain.Notification{
		UserID:    userID,
		Channel:   domain.NotificationChannelEmail,
		EventType: eventType,
		Subject:   subject,
		Body:      body,
		Payload:   payload,
		Status:    domain.NotificationStatusPending,
	}
	return w.notifications.Enqueue(ctx, n)
}
