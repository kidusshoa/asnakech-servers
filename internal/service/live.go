package service

import (
	"context"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/live"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

const (
	checkInEarlyWindow = 15 * time.Minute
	checkInLateGrace   = 10 * time.Minute
	checkInEndGrace    = 30 * time.Minute
)

type LiveService struct {
	courses     repository.CourseRepository
	enrollments repository.EnrollmentRepository
	sessions    repository.LiveSessionRepository
	attendance  repository.SessionAttendanceRepository
	providers   *live.Registry
}

func NewLiveService(
	courses repository.CourseRepository,
	enrollments repository.EnrollmentRepository,
	sessions repository.LiveSessionRepository,
	attendance repository.SessionAttendanceRepository,
	providers *live.Registry,
) *LiveService {
	return &LiveService{
		courses:     courses,
		enrollments: enrollments,
		sessions:    sessions,
		attendance:  attendance,
		providers:   providers,
	}
}

func (s *LiveService) CreateSession(ctx context.Context, actorID, courseID string, in domain.LiveSessionCreate, platformAdmin bool) (*domain.LiveSession, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	if in.EndsAt.Before(in.StartsAt) || in.EndsAt.Equal(in.StartsAt) {
		return nil, apperr.Validation("ends_at must be after starts_at")
	}
	provider := in.Provider
	if provider == "" {
		provider = s.providers.DefaultProvider()
	}
	if !validProvider(provider) {
		return nil, apperr.Validation("invalid provider")
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz == "" {
		tz = "UTC"
	}
	session := &domain.LiveSession{
		CourseID:         courseID,
		LessonID:         in.LessonID,
		Title:            title,
		Description:      strings.TrimSpace(in.Description),
		Status:           domain.LiveSessionStatusDraft,
		StartsAt:         in.StartsAt.UTC(),
		EndsAt:           in.EndsAt.UTC(),
		Timezone:         tz,
		Provider:         provider,
		JoinURL:          strings.TrimSpace(in.JoinURL),
		HostURL:          strings.TrimSpace(in.HostURL),
		ProviderMetadata: map[string]string{},
		CreatedBy:        actorID,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *LiveService) ListCourseSessions(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.LiveSession, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	items, err := s.sessions.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if canAuthor {
		return items, nil
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, courseID); err != nil {
		return nil, err
	}
	out := make([]domain.LiveSession, 0, len(items))
	for _, item := range items {
		if item.Status == domain.LiveSessionStatusScheduled || item.Status == domain.LiveSessionStatusCompleted {
			out = append(out, stripSessionLinks(item))
		}
	}
	return out, nil
}

func (s *LiveService) GetSession(ctx context.Context, actorID, sessionID string, platformAdmin bool) (*domain.LiveSession, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, session.CourseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	if err := s.requireSessionRead(ctx, actorID, session, platformAdmin); err != nil {
		return nil, err
	}
	if !canAuthor {
		stripped := stripSessionLinks(*session)
		return &stripped, nil
	}
	return session, nil
}

func (s *LiveService) UpdateSession(ctx context.Context, actorID, sessionID string, patch domain.LiveSessionUpdate, platformAdmin bool) (*domain.LiveSession, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, session.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if session.Status == domain.LiveSessionStatusCancelled {
		return nil, apperr.Validation("cancelled sessions cannot be edited")
	}
	if patch.LessonID != nil {
		session.LessonID = patch.LessonID
	}
	if patch.Title != nil {
		t := strings.TrimSpace(*patch.Title)
		if t == "" {
			return nil, apperr.Validation("title cannot be empty")
		}
		session.Title = t
	}
	if patch.Description != nil {
		session.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.StartsAt != nil {
		session.StartsAt = patch.StartsAt.UTC()
	}
	if patch.EndsAt != nil {
		session.EndsAt = patch.EndsAt.UTC()
	}
	if session.EndsAt.Before(session.StartsAt) || session.EndsAt.Equal(session.StartsAt) {
		return nil, apperr.Validation("ends_at must be after starts_at")
	}
	if patch.Timezone != nil {
		tz := strings.TrimSpace(*patch.Timezone)
		if tz == "" {
			tz = "UTC"
		}
		session.Timezone = tz
	}
	if patch.Provider != nil {
		if !validProvider(*patch.Provider) {
			return nil, apperr.Validation("invalid provider")
		}
		session.Provider = *patch.Provider
	}
	if patch.JoinURL != nil {
		session.JoinURL = strings.TrimSpace(*patch.JoinURL)
	}
	if patch.HostURL != nil {
		session.HostURL = strings.TrimSpace(*patch.HostURL)
	}
	return s.sessions.Update(ctx, session)
}

func (s *LiveService) PublishSession(ctx context.Context, actorID, sessionID string, platformAdmin bool) (*domain.LiveSession, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, session.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if session.Status != domain.LiveSessionStatusDraft {
		return nil, apperr.Validation("only draft sessions can be published")
	}
	if session.JoinURL == "" && session.Provider != domain.LiveProviderCustom {
		if err := s.applyGeneratedLink(ctx, session); err != nil {
			return nil, err
		}
	}
	if session.JoinURL == "" {
		return nil, apperr.Validation("join_url is required before publishing")
	}
	return s.sessions.SetStatus(ctx, sessionID, domain.LiveSessionStatusScheduled)
}

func (s *LiveService) CompleteSession(ctx context.Context, actorID, sessionID string, platformAdmin bool) (*domain.LiveSession, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, session.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if session.Status != domain.LiveSessionStatusScheduled {
		return nil, apperr.Validation("only scheduled sessions can be completed")
	}
	return s.sessions.SetStatus(ctx, sessionID, domain.LiveSessionStatusCompleted)
}

func (s *LiveService) CancelSession(ctx context.Context, actorID, sessionID string, platformAdmin bool) (*domain.LiveSession, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, session.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if session.Status == domain.LiveSessionStatusCompleted {
		return nil, apperr.Validation("completed sessions cannot be cancelled")
	}
	return s.sessions.SetStatus(ctx, sessionID, domain.LiveSessionStatusCancelled)
}

func (s *LiveService) GenerateLink(ctx context.Context, actorID, sessionID string, platformAdmin bool) (*domain.LiveSession, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, session.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if err := s.applyGeneratedLink(ctx, session); err != nil {
		return nil, err
	}
	return s.sessions.Update(ctx, session)
}

func (s *LiveService) JoinSession(ctx context.Context, actorID, sessionID string, platformAdmin bool) (*domain.JoinInfo, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, session.CourseID)
	if err != nil {
		return nil, err
	}
	isHost := platformAdmin || course.TeacherID == actorID
	if !isHost {
		if session.Status != domain.LiveSessionStatusScheduled {
			return nil, apperr.Forbidden("session is not open for joining")
		}
		if _, err := s.requireActiveEnrollment(ctx, actorID, session.CourseID); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(session.JoinURL) == "" {
		return nil, apperr.Validation("session has no join link")
	}
	info := &domain.JoinInfo{
		SessionID: session.ID,
		JoinURL:   session.JoinURL,
		StartsAt:  session.StartsAt,
		EndsAt:    session.EndsAt,
		Status:    session.Status,
		IsHost:    isHost,
	}
	if isHost {
		info.HostURL = session.HostURL
		if info.HostURL == "" {
			info.HostURL = session.JoinURL
		}
	}
	return info, nil
}

func (s *LiveService) ListAttendance(ctx context.Context, actorID, sessionID string, platformAdmin bool) ([]domain.SessionAttendance, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, session.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.attendance.ListBySession(ctx, sessionID)
}

func (s *LiveService) MarkAttendance(ctx context.Context, actorID, sessionID, userID string, status domain.AttendanceStatus, note string, platformAdmin bool) (*domain.SessionAttendance, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, session.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if !validAttendanceStatus(status) {
		return nil, apperr.Validation("invalid attendance status")
	}
	now := time.Now().UTC()
	a := &domain.SessionAttendance{
		SessionID: sessionID,
		UserID:    userID,
		Status:    status,
		MarkedBy:  &actorID,
		Note:      strings.TrimSpace(note),
	}
	if status == domain.AttendanceStatusPresent || status == domain.AttendanceStatusLate {
		a.JoinedAt = &now
	}
	if err := s.attendance.Upsert(ctx, a); err != nil {
		return nil, err
	}
	return s.attendance.GetBySessionUser(ctx, sessionID, userID)
}

func (s *LiveService) CheckIn(ctx context.Context, actorID, sessionID string, platformAdmin bool) (*domain.SessionAttendance, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, session.CourseID)
	if err != nil {
		return nil, err
	}
	isHost := platformAdmin || course.TeacherID == actorID
	if !isHost {
		if session.Status != domain.LiveSessionStatusScheduled {
			return nil, apperr.Forbidden("session is not open for check-in")
		}
		if _, err := s.requireActiveEnrollment(ctx, actorID, session.CourseID); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	windowStart := session.StartsAt.Add(-checkInEarlyWindow)
	windowEnd := session.EndsAt.Add(checkInEndGrace)
	if now.Before(windowStart) || now.After(windowEnd) {
		return nil, apperr.Forbidden("check-in is outside the allowed window")
	}
	status := domain.AttendanceStatusPresent
	if now.After(session.StartsAt.Add(checkInLateGrace)) {
		status = domain.AttendanceStatusLate
	}
	a := &domain.SessionAttendance{
		SessionID: sessionID,
		UserID:    actorID,
		Status:    status,
		JoinedAt:  &now,
	}
	if err := s.attendance.Upsert(ctx, a); err != nil {
		return nil, err
	}
	return s.attendance.GetBySessionUser(ctx, sessionID, actorID)
}

func (s *LiveService) ListCalendar(ctx context.Context, actorID string, from, to time.Time, page, perPage int, platformAdmin bool) ([]domain.LiveSession, int, error) {
	if to.Before(from) {
		return nil, 0, apperr.Validation("to must be after from")
	}
	if to.Sub(from) > 366*24*time.Hour {
		return nil, 0, apperr.Validation("date range cannot exceed 366 days")
	}
	items, total, err := s.sessions.ListCalendar(ctx, domain.CalendarFilter{
		From:    from.UTC(),
		To:      to.UTC(),
		UserID:  actorID,
		Admin:   platformAdmin,
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return nil, 0, err
	}
	if platformAdmin {
		return items, total, nil
	}
	out := make([]domain.LiveSession, 0, len(items))
	for _, item := range items {
		course, err := s.courses.GetByID(ctx, item.CourseID)
		if err != nil {
			return nil, 0, err
		}
		if course.TeacherID == actorID {
			out = append(out, item)
		} else {
			out = append(out, stripSessionLinks(item))
		}
	}
	return out, total, nil
}

func (s *LiveService) applyGeneratedLink(ctx context.Context, session *domain.LiveSession) error {
	link, err := s.providers.For(session.Provider).GenerateLink(ctx, session)
	if err != nil {
		return err
	}
	session.JoinURL = link.JoinURL
	if link.HostURL != "" {
		session.HostURL = link.HostURL
	}
	session.ExternalID = link.ExternalID
	if len(link.Metadata) > 0 {
		session.ProviderMetadata = link.Metadata
	}
	_, err = s.sessions.Update(ctx, session)
	return err
}

func (s *LiveService) requireCourseWrite(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	return apperr.Forbidden("only the course teacher or an admin can manage live sessions")
}

func (s *LiveService) requireActiveEnrollment(ctx context.Context, userID, courseID string) (*domain.Enrollment, error) {
	en, err := s.enrollments.GetByCourseUser(ctx, courseID, userID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.Forbidden("active enrollment required")
		}
		return nil, err
	}
	if en.Status != domain.EnrollmentStatusActive {
		return nil, apperr.Forbidden("active enrollment required")
	}
	return en, nil
}

func (s *LiveService) requireSessionRead(ctx context.Context, actorID string, session *domain.LiveSession, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, session.CourseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	if session.Status == domain.LiveSessionStatusDraft || session.Status == domain.LiveSessionStatusCancelled {
		return apperr.NotFound("session not found")
	}
	_, err = s.requireActiveEnrollment(ctx, actorID, session.CourseID)
	return err
}

func validProvider(p domain.LiveProvider) bool {
	switch p {
	case domain.LiveProviderCustom, domain.LiveProviderJitsi, domain.LiveProviderZoom, domain.LiveProviderGoogleMeet:
		return true
	default:
		return false
	}
}

func validAttendanceStatus(s domain.AttendanceStatus) bool {
	switch s {
	case domain.AttendanceStatusRegistered, domain.AttendanceStatusPresent,
		domain.AttendanceStatusAbsent, domain.AttendanceStatusLate, domain.AttendanceStatusExcused:
		return true
	default:
		return false
	}
}

func stripSessionLinks(s domain.LiveSession) domain.LiveSession {
	s.JoinURL = ""
	s.HostURL = ""
	s.ExternalID = ""
	s.ProviderMetadata = nil
	return s
}
