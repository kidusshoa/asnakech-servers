package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type LiveSessionRepository interface {
	Create(ctx context.Context, s *domain.LiveSession) error
	GetByID(ctx context.Context, id string) (*domain.LiveSession, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.LiveSession, error)
	Update(ctx context.Context, s *domain.LiveSession) (*domain.LiveSession, error)
	SetStatus(ctx context.Context, id string, status domain.LiveSessionStatus) (*domain.LiveSession, error)
	ListCalendar(ctx context.Context, filter domain.CalendarFilter) ([]domain.LiveSession, int, error)
}

type SessionAttendanceRepository interface {
	Upsert(ctx context.Context, a *domain.SessionAttendance) error
	GetBySessionUser(ctx context.Context, sessionID, userID string) (*domain.SessionAttendance, error)
	ListBySession(ctx context.Context, sessionID string) ([]domain.SessionAttendance, error)
}
