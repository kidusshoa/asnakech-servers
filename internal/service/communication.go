package service

import (
	"context"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/notify"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type CommunicationService struct {
	courses       repository.CourseRepository
	enrollments   repository.EnrollmentRepository
	users         repository.UserRepository
	announcements repository.AnnouncementRepository
	threads       repository.DiscussionThreadRepository
	posts         repository.DiscussionPostRepository
	conversations repository.DMConversationRepository
	messages      repository.DMMessageRepository
	notifications repository.NotificationRepository
	outbox        *notify.OutboxWriter
}

func NewCommunicationService(
	courses repository.CourseRepository,
	enrollments repository.EnrollmentRepository,
	users repository.UserRepository,
	announcements repository.AnnouncementRepository,
	threads repository.DiscussionThreadRepository,
	posts repository.DiscussionPostRepository,
	conversations repository.DMConversationRepository,
	messages repository.DMMessageRepository,
	notifications repository.NotificationRepository,
	outbox *notify.OutboxWriter,
) *CommunicationService {
	return &CommunicationService{
		courses:       courses,
		enrollments:   enrollments,
		users:         users,
		announcements: announcements,
		threads:       threads,
		posts:         posts,
		conversations: conversations,
		messages:      messages,
		notifications: notifications,
		outbox:        outbox,
	}
}

// --- Announcements ---

func (s *CommunicationService) CreateAnnouncement(ctx context.Context, actorID, courseID, title, body string, pinned bool, platformAdmin bool) (*domain.Announcement, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	a := &domain.Announcement{
		CourseID: courseID,
		AuthorID: actorID,
		Title:    title,
		Body:     strings.TrimSpace(body),
		Status:   domain.AnnouncementStatusDraft,
		Pinned:   pinned,
	}
	if err := s.announcements.Create(ctx, a); err != nil {
		return nil, err
	}
	a.AuthorName, _ = s.userName(ctx, actorID)
	return a, nil
}

func (s *CommunicationService) ListAnnouncements(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.Announcement, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	if !canAuthor {
		if _, err := s.requireActiveEnrollment(ctx, actorID, courseID); err != nil {
			return nil, err
		}
	}
	return s.announcements.ListByCourse(ctx, courseID, canAuthor)
}

func (s *CommunicationService) GetAnnouncement(ctx context.Context, actorID, id string, platformAdmin bool) (*domain.Announcement, error) {
	a, err := s.announcements.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireAnnouncementRead(ctx, actorID, a, platformAdmin); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *CommunicationService) UpdateAnnouncement(ctx context.Context, actorID, id, title, body string, pinned bool, platformAdmin bool) (*domain.Announcement, error) {
	a, err := s.announcements.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, a.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	a.Title = title
	a.Body = strings.TrimSpace(body)
	a.Pinned = pinned
	return s.announcements.Update(ctx, a)
}

func (s *CommunicationService) PublishAnnouncement(ctx context.Context, actorID, id string, platformAdmin bool) (*domain.Announcement, error) {
	a, err := s.announcements.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, a.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if a.Status != domain.AnnouncementStatusDraft {
		return nil, apperr.Validation("only draft announcements can be published")
	}
	if err := s.announcements.SetStatus(ctx, id, domain.AnnouncementStatusPublished, nil); err != nil {
		return nil, err
	}
	a, _ = s.announcements.GetByID(ctx, id)
	s.notifyCourseEnrollees(ctx, a.CourseID, actorID, "announcement.published", a.Title, a.Body, map[string]string{
		"announcement_id": a.ID,
		"course_id":       a.CourseID,
	})
	return a, nil
}

func (s *CommunicationService) DeleteAnnouncement(ctx context.Context, actorID, id string, platformAdmin bool) error {
	a, err := s.announcements.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, a.CourseID, platformAdmin); err != nil {
		return err
	}
	return s.announcements.Delete(ctx, id)
}

// --- Discussion threads ---

func (s *CommunicationService) CreateThread(ctx context.Context, actorID, courseID, title, body string, platformAdmin bool) (*domain.DiscussionThread, error) {
	if err := s.requireCourseRead(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	if body == "" {
		return nil, apperr.Validation("body is required")
	}
	t := &domain.DiscussionThread{
		CourseID: courseID,
		AuthorID: actorID,
		Title:    title,
		Status:   domain.ThreadStatusOpen,
	}
	if err := s.threads.Create(ctx, t); err != nil {
		return nil, err
	}
	p := &domain.DiscussionPost{ThreadID: t.ID, AuthorID: actorID, Body: body}
	if err := s.posts.Create(ctx, p); err != nil {
		return nil, err
	}
	t, _ = s.threads.GetByID(ctx, t.ID)
	s.notifyCourseTeacher(ctx, courseID, actorID, "discussion.new_thread", t.Title, body, map[string]string{
		"thread_id": t.ID,
		"course_id": courseID,
	})
	return t, nil
}

func (s *CommunicationService) ListThreads(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.DiscussionThread, error) {
	if err := s.requireCourseRead(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.threads.ListByCourse(ctx, courseID)
}

func (s *CommunicationService) GetThread(ctx context.Context, actorID, threadID string, platformAdmin bool) (*domain.DiscussionThread, error) {
	t, err := s.threads.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseRead(ctx, actorID, t.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *CommunicationService) LockThread(ctx context.Context, actorID, threadID string, platformAdmin bool) (*domain.DiscussionThread, error) {
	t, err := s.threads.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, t.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.threads.SetStatus(ctx, threadID, domain.ThreadStatusLocked)
}

func (s *CommunicationService) CreatePost(ctx context.Context, actorID, threadID, body string, parentID *string, platformAdmin bool) (*domain.DiscussionPost, error) {
	t, err := s.threads.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseRead(ctx, actorID, t.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if t.Status == domain.ThreadStatusLocked {
		return nil, apperr.Forbidden("thread is locked")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, apperr.Validation("body is required")
	}
	p := &domain.DiscussionPost{ThreadID: threadID, AuthorID: actorID, ParentID: parentID, Body: body}
	if err := s.posts.Create(ctx, p); err != nil {
		return nil, err
	}
	p, _ = s.posts.GetByID(ctx, p.ID)
	s.notifyThreadParticipants(ctx, threadID, actorID, t.Title, body, t.CourseID, t.AuthorID)
	return p, nil
}

func (s *CommunicationService) ListPosts(ctx context.Context, actorID, threadID string, platformAdmin bool) ([]domain.DiscussionPost, error) {
	t, err := s.threads.GetByID(ctx, threadID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseRead(ctx, actorID, t.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.posts.ListByThread(ctx, threadID)
}

func (s *CommunicationService) UpdatePost(ctx context.Context, actorID, postID, body string, platformAdmin bool) (*domain.DiscussionPost, error) {
	p, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	t, err := s.threads.GetByID(ctx, p.ThreadID)
	if err != nil {
		return nil, err
	}
	if p.AuthorID != actorID {
		if err := s.requireCourseWrite(ctx, actorID, t.CourseID, platformAdmin); err != nil {
			return nil, apperr.Forbidden("only the author or teacher can edit this post")
		}
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, apperr.Validation("body is required")
	}
	p.Body = body
	return s.posts.Update(ctx, p)
}

func (s *CommunicationService) DeletePost(ctx context.Context, actorID, postID string, platformAdmin bool) error {
	p, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	t, err := s.threads.GetByID(ctx, p.ThreadID)
	if err != nil {
		return err
	}
	if p.AuthorID != actorID {
		if err := s.requireCourseWrite(ctx, actorID, t.CourseID, platformAdmin); err != nil {
			return apperr.Forbidden("only the author or teacher can delete this post")
		}
	}
	return s.posts.Delete(ctx, postID)
}

// --- Direct messages ---

func (s *CommunicationService) StartConversation(ctx context.Context, actorID, otherUserID string) (*domain.DMConversation, error) {
	otherUserID = strings.TrimSpace(otherUserID)
	if otherUserID == "" || otherUserID == actorID {
		return nil, apperr.Validation("invalid recipient")
	}
	if _, err := s.users.GetByID(ctx, otherUserID); err != nil {
		return nil, err
	}
	c, err := s.conversations.GetOrCreate(ctx, actorID, otherUserID)
	if err != nil {
		return nil, err
	}
	return s.decorateConversation(c, actorID), nil
}

func (s *CommunicationService) ListConversations(ctx context.Context, actorID string) ([]domain.DMConversation, error) {
	return s.conversations.ListForUser(ctx, actorID)
}

func (s *CommunicationService) SendMessage(ctx context.Context, actorID, conversationID, body string) (*domain.DMMessage, error) {
	c, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if c.UserAID != actorID && c.UserBID != actorID {
		return nil, apperr.Forbidden("not a participant")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, apperr.Validation("body is required")
	}
	m := &domain.DMMessage{ConversationID: conversationID, SenderID: actorID, Body: body}
	if err := s.messages.Create(ctx, m); err != nil {
		return nil, err
	}
	recipient := c.UserBID
	if recipient == actorID {
		recipient = c.UserAID
	}
	_ = s.outbox.EnqueueInApp(ctx, recipient, "dm.message", "New message", body, map[string]string{
		"conversation_id": conversationID,
		"sender_id":       actorID,
	})
	return m, nil
}

func (s *CommunicationService) ListMessages(ctx context.Context, actorID, conversationID string, page, perPage int) ([]domain.DMMessage, int, error) {
	c, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		return nil, 0, err
	}
	if c.UserAID != actorID && c.UserBID != actorID {
		return nil, 0, apperr.Forbidden("not a participant")
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage
	return s.messages.ListByConversation(ctx, conversationID, perPage, offset)
}

func (s *CommunicationService) MarkConversationRead(ctx context.Context, actorID, conversationID string) error {
	c, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		return err
	}
	if c.UserAID != actorID && c.UserBID != actorID {
		return apperr.Forbidden("not a participant")
	}
	return s.messages.MarkRead(ctx, conversationID, actorID)
}

// --- Notifications ---

func (s *CommunicationService) ListNotifications(ctx context.Context, actorID string, filter domain.NotificationListFilter) ([]domain.Notification, int, error) {
	return s.notifications.ListForUser(ctx, actorID, filter)
}

func (s *CommunicationService) MarkNotificationRead(ctx context.Context, actorID, id string) error {
	return s.notifications.MarkRead(ctx, id, actorID)
}

func (s *CommunicationService) MarkAllNotificationsRead(ctx context.Context, actorID string) error {
	return s.notifications.MarkAllRead(ctx, actorID)
}

// --- helpers ---

func (s *CommunicationService) requireCourseWrite(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	return apperr.Forbidden("only the course teacher or an admin can manage course content")
}

func (s *CommunicationService) requireCourseRead(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	_, err = s.requireActiveEnrollment(ctx, actorID, courseID)
	return err
}

func (s *CommunicationService) requireActiveEnrollment(ctx context.Context, userID, courseID string) (*domain.Enrollment, error) {
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

func (s *CommunicationService) requireAnnouncementRead(ctx context.Context, actorID string, a *domain.Announcement, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, a.CourseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	if a.Status != domain.AnnouncementStatusPublished {
		return apperr.NotFound("announcement not found")
	}
	_, err = s.requireActiveEnrollment(ctx, actorID, a.CourseID)
	return err
}

func (s *CommunicationService) notifyCourseEnrollees(ctx context.Context, courseID, excludeID, eventType, subject, body string, payload map[string]string) {
	items, _, err := s.enrollments.ListByCourse(ctx, courseID, domain.EnrollmentListFilter{PerPage: 1000, Status: domain.EnrollmentStatusActive})
	if err != nil {
		return
	}
	for _, en := range items {
		if en.UserID == excludeID {
			continue
		}
		_ = s.outbox.EnqueueInApp(ctx, en.UserID, eventType, subject, body, payload)
	}
}

func (s *CommunicationService) notifyCourseTeacher(ctx context.Context, courseID, excludeID, eventType, subject, body string, payload map[string]string) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil || course.TeacherID == excludeID {
		return
	}
	_ = s.outbox.EnqueueInApp(ctx, course.TeacherID, eventType, subject, body, payload)
}

func (s *CommunicationService) notifyThreadParticipants(ctx context.Context, threadID, excludeID, subject, body, courseID, threadAuthorID string) {
	ids, err := s.posts.ListParticipantIDs(ctx, threadID)
	if err != nil {
		return
	}
	seen := map[string]struct{}{excludeID: {}}
	course, _ := s.courses.GetByID(ctx, courseID)
	if course != nil {
		seen[course.TeacherID] = struct{}{}
	}
	seen[threadAuthorID] = struct{}{}
	for _, id := range ids {
		if _, ok := seen[id]; ok || id == excludeID {
			continue
		}
		seen[id] = struct{}{}
		_ = s.outbox.EnqueueInApp(ctx, id, "discussion.reply", subject, body, map[string]string{
			"thread_id": threadID,
			"course_id": courseID,
		})
	}
	if course != nil && course.TeacherID != excludeID {
		_ = s.outbox.EnqueueInApp(ctx, course.TeacherID, "discussion.reply", subject, body, map[string]string{
			"thread_id": threadID,
			"course_id": courseID,
		})
	}
}

func (s *CommunicationService) userName(ctx context.Context, userID string) (string, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.FullName, nil
}

func (s *CommunicationService) decorateConversation(c *domain.DMConversation, actorID string) *domain.DMConversation {
	if c.UserAID == actorID {
		c.OtherUserID = c.UserBID
	} else {
		c.OtherUserID = c.UserAID
	}
	return c
}
