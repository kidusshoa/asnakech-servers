package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type EnrollmentService struct {
	courses     repository.CourseRepository
	enrollments repository.EnrollmentRepository
	invites     repository.EnrollmentInviteCodeRepository
}

func NewEnrollmentService(
	courses repository.CourseRepository,
	enrollments repository.EnrollmentRepository,
	invites repository.EnrollmentInviteCodeRepository,
) *EnrollmentService {
	return &EnrollmentService{
		courses:     courses,
		enrollments: enrollments,
		invites:     invites,
	}
}

func (s *EnrollmentService) Enroll(ctx context.Context, actorID, courseID, inviteCode string) (*domain.Enrollment, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course.Status != domain.CourseStatusPublished {
		return nil, apperr.Validation("only published courses accept enrollment")
	}
	if course.TeacherID == actorID {
		return nil, apperr.Validation("teachers cannot enroll in their own course")
	}

	inviteCode = strings.TrimSpace(inviteCode)
	var invite *domain.EnrollmentInviteCode
	if inviteCode != "" {
		invite, err = s.invites.GetByCourseCode(ctx, courseID, inviteCode)
		if err != nil {
			return nil, err
		}
		if err := validateInvite(invite); err != nil {
			return nil, err
		}
	} else if !course.EnrollmentOpen {
		return nil, apperr.Forbidden("enrollment is closed; an invite code is required")
	}

	if existing, err := s.enrollments.GetByCourseUser(ctx, courseID, actorID); err == nil {
		switch existing.Status {
		case domain.EnrollmentStatusActive, domain.EnrollmentStatusWaitlisted:
			return nil, apperr.Conflict("already enrolled in this course")
		case domain.EnrollmentStatusCancelled:
			return s.reactivate(ctx, course, existing, invite)
		}
	} else if !apperr.IsNotFound(err) {
		return nil, err
	}

	now := time.Now().UTC()
	status, eventType, err := s.decideSeat(ctx, course)
	if err != nil {
		return nil, err
	}

	en := &domain.Enrollment{
		CourseID: courseID,
		UserID:   actorID,
		Status:   status,
		Source:   domain.EnrollmentSourceSelf,
	}
	if invite != nil {
		en.Source = domain.EnrollmentSourceInviteCode
		en.InviteCodeID = &invite.ID
	}
	if status == domain.EnrollmentStatusActive {
		en.EnrolledAt = &now
	} else {
		en.WaitlistedAt = &now
	}

	if err := s.enrollments.Create(ctx, en); err != nil {
		return nil, err
	}
	if invite != nil {
		if err := s.invites.IncrementUses(ctx, invite.ID); err != nil {
			return nil, err
		}
	}
	if err := s.emit(ctx, en, eventType); err != nil {
		return nil, err
	}
	return s.enrollments.GetByID(ctx, en.ID)
}

func (s *EnrollmentService) reactivate(ctx context.Context, course *domain.Course, existing *domain.Enrollment, invite *domain.EnrollmentInviteCode) (*domain.Enrollment, error) {
	status, eventType, err := s.decideSeat(ctx, course)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	existing.Status = status
	existing.CancelledAt = nil
	existing.Source = domain.EnrollmentSourceSelf
	if invite != nil {
		existing.Source = domain.EnrollmentSourceInviteCode
		existing.InviteCodeID = &invite.ID
	}
	if status == domain.EnrollmentStatusActive {
		existing.EnrolledAt = &now
		existing.WaitlistedAt = nil
	} else {
		existing.WaitlistedAt = &now
		existing.EnrolledAt = nil
	}
	if err := s.enrollments.UpdateStatus(ctx, existing); err != nil {
		return nil, err
	}
	if invite != nil {
		if err := s.invites.IncrementUses(ctx, invite.ID); err != nil {
			return nil, err
		}
	}
	if err := s.emit(ctx, existing, eventType); err != nil {
		return nil, err
	}
	return s.enrollments.GetByID(ctx, existing.ID)
}

func (s *EnrollmentService) Unenroll(ctx context.Context, actorID, courseID string, platformAdmin bool) (*domain.Enrollment, error) {
	en, err := s.enrollments.GetByCourseUser(ctx, courseID, actorID)
	if err != nil {
		return nil, err
	}
	if en.UserID != actorID && !platformAdmin {
		return nil, apperr.Forbidden("cannot unenroll another user")
	}
	if en.Status == domain.EnrollmentStatusCancelled {
		return nil, apperr.Conflict("enrollment already cancelled")
	}

	wasActive := en.Status == domain.EnrollmentStatusActive
	now := time.Now().UTC()
	en.Status = domain.EnrollmentStatusCancelled
	en.CancelledAt = &now
	if err := s.enrollments.UpdateStatus(ctx, en); err != nil {
		return nil, err
	}
	if err := s.emit(ctx, en, domain.EnrollmentEventCancelled); err != nil {
		return nil, err
	}

	if wasActive {
		if err := s.promoteWaitlist(ctx, courseID); err != nil {
			return nil, err
		}
	}
	return s.enrollments.GetByID(ctx, en.ID)
}

func (s *EnrollmentService) promoteWaitlist(ctx context.Context, courseID string) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if !course.WaitlistEnabled {
		return nil
	}
	if course.EnrollmentCapacity != nil {
		active, err := s.enrollments.CountByCourseStatus(ctx, courseID, domain.EnrollmentStatusActive)
		if err != nil {
			return err
		}
		if int(active) >= *course.EnrollmentCapacity {
			return nil
		}
	}

	next, err := s.enrollments.NextWaitlisted(ctx, courseID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil
		}
		return err
	}
	now := time.Now().UTC()
	next.Status = domain.EnrollmentStatusActive
	next.EnrolledAt = &now
	if err := s.enrollments.UpdateStatus(ctx, next); err != nil {
		return err
	}
	return s.emit(ctx, next, domain.EnrollmentEventActivated)
}

func (s *EnrollmentService) ListMine(ctx context.Context, actorID string, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int64, error) {
	return s.enrollments.ListByUser(ctx, actorID, filter)
}

func (s *EnrollmentService) ListForCourse(ctx context.Context, actorID, courseID string, filter domain.EnrollmentListFilter, platformAdmin bool) ([]domain.Enrollment, int64, error) {
	if err := s.requireCourseManage(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, 0, err
	}
	return s.enrollments.ListByCourse(ctx, courseID, filter)
}

func (s *EnrollmentService) GetAccess(ctx context.Context, actorID, courseID string, platformAdmin bool) (*domain.CourseAccess, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	access := &domain.CourseAccess{
		CourseID:        courseID,
		IsTeacher:       actorID != "" && course.TeacherID == actorID,
		IsPlatformAdmin: platformAdmin,
	}
	if access.IsTeacher || access.IsPlatformAdmin {
		access.CanAccessContent = true
		return access, nil
	}
	if actorID == "" {
		return access, nil
	}
	en, err := s.enrollments.GetByCourseUser(ctx, courseID, actorID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return access, nil
		}
		return nil, err
	}
	access.Enrollment = en
	access.CanAccessContent = en.Status == domain.EnrollmentStatusActive
	return access, nil
}

func (s *EnrollmentService) HasActiveEnrollment(ctx context.Context, courseID, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	en, err := s.enrollments.GetByCourseUser(ctx, courseID, userID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return en.Status == domain.EnrollmentStatusActive, nil
}

func (s *EnrollmentService) UpdateSettings(ctx context.Context, actorID, courseID string, capacity *int, open *bool, waitlist *bool, platformAdmin bool) (*domain.Course, error) {
	if err := s.requireCourseManage(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	settings := domain.CourseEnrollmentSettings{
		Capacity:        course.EnrollmentCapacity,
		EnrollmentOpen:  course.EnrollmentOpen,
		WaitlistEnabled: course.WaitlistEnabled,
	}
	if capacity != nil {
		if *capacity <= 0 {
			// treat 0 / negative sentinel as clear unlimited? Use explicit null via negative? Better: pointer to clear — allow capacity=0 to mean unlimited by converting
			settings.Capacity = nil
		} else {
			settings.Capacity = capacity
		}
	}
	if open != nil {
		settings.EnrollmentOpen = *open
	}
	if waitlist != nil {
		settings.WaitlistEnabled = *waitlist
	}
	return s.courses.UpdateEnrollmentSettings(ctx, courseID, settings)
}

func (s *EnrollmentService) CreateInviteCode(ctx context.Context, actorID, courseID, code string, maxUses *int, expiresAt *time.Time, platformAdmin bool) (*domain.EnrollmentInviteCode, error) {
	if err := s.requireCourseManage(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		raw := make([]byte, 4)
		if _, err := rand.Read(raw); err != nil {
			return nil, apperr.Internal("failed to generate invite code")
		}
		code = strings.ToUpper(hex.EncodeToString(raw))
	}
	if maxUses != nil && *maxUses <= 0 {
		return nil, apperr.Validation("max_uses must be > 0 when set")
	}
	if expiresAt != nil && expiresAt.Before(time.Now().UTC()) {
		return nil, apperr.Validation("expires_at must be in the future")
	}
	inv := &domain.EnrollmentInviteCode{
		CourseID:  courseID,
		Code:      code,
		MaxUses:   maxUses,
		ExpiresAt: expiresAt,
		CreatedBy: actorID,
	}
	if err := s.invites.Create(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *EnrollmentService) ListInviteCodes(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.EnrollmentInviteCode, error) {
	if err := s.requireCourseManage(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.invites.ListByCourse(ctx, courseID)
}

func (s *EnrollmentService) RevokeInviteCode(ctx context.Context, actorID, courseID, codeID string, platformAdmin bool) error {
	if err := s.requireCourseManage(ctx, actorID, courseID, platformAdmin); err != nil {
		return err
	}
	inv, err := s.invites.GetByID(ctx, codeID)
	if err != nil {
		return err
	}
	if inv.CourseID != courseID {
		return apperr.NotFound("invite code not found")
	}
	return s.invites.Revoke(ctx, codeID)
}

func (s *EnrollmentService) decideSeat(ctx context.Context, course *domain.Course) (domain.EnrollmentStatus, domain.EnrollmentEventType, error) {
	if course.EnrollmentCapacity == nil {
		return domain.EnrollmentStatusActive, domain.EnrollmentEventEnrolled, nil
	}
	active, err := s.enrollments.CountByCourseStatus(ctx, course.ID, domain.EnrollmentStatusActive)
	if err != nil {
		return "", "", err
	}
	if int(active) < *course.EnrollmentCapacity {
		return domain.EnrollmentStatusActive, domain.EnrollmentEventEnrolled, nil
	}
	if course.WaitlistEnabled {
		return domain.EnrollmentStatusWaitlisted, domain.EnrollmentEventWaitlisted, nil
	}
	return "", "", apperr.Conflict("course is full")
}

func (s *EnrollmentService) emit(ctx context.Context, en *domain.Enrollment, typ domain.EnrollmentEventType) error {
	return s.enrollments.AppendEvent(ctx, &domain.EnrollmentEvent{
		EnrollmentID: en.ID,
		CourseID:     en.CourseID,
		UserID:       en.UserID,
		EventType:    typ,
	})
}

func (s *EnrollmentService) requireCourseManage(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	return apperr.Forbidden("only the course teacher or an admin can manage enrollments")
}

func validateInvite(inv *domain.EnrollmentInviteCode) error {
	if inv.RevokedAt != nil {
		return apperr.Forbidden("invite code has been revoked")
	}
	if inv.ExpiresAt != nil && inv.ExpiresAt.Before(time.Now().UTC()) {
		return apperr.Forbidden("invite code has expired")
	}
	if inv.MaxUses != nil && inv.UsesCount >= *inv.MaxUses {
		return apperr.Forbidden("invite code has reached its use limit")
	}
	return nil
}
