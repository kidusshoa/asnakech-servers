package service

import (
	"context"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type ProgressService struct {
	modules        repository.ModuleRepository
	lessons        repository.LessonRepository
	enrollments    repository.EnrollmentRepository
	lessonProgress repository.LessonProgressRepository
	courseProgress repository.CourseProgressRepository
}

func NewProgressService(
	modules repository.ModuleRepository,
	lessons repository.LessonRepository,
	enrollments repository.EnrollmentRepository,
	lessonProgress repository.LessonProgressRepository,
	courseProgress repository.CourseProgressRepository,
) *ProgressService {
	return &ProgressService{
		modules:        modules,
		lessons:        lessons,
		enrollments:    enrollments,
		lessonProgress: lessonProgress,
		courseProgress: courseProgress,
	}
}

func (s *ProgressService) UpsertLessonProgress(ctx context.Context, actorID, lessonID string, in domain.LessonProgressUpsert) (*domain.LessonProgress, *domain.CourseProgress, error) {
	lesson, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return nil, nil, err
	}
	enrollment, err := s.requireActiveEnrollment(ctx, actorID, courseID)
	if err != nil {
		return nil, nil, err
	}
	if lesson.Status != domain.LessonStatusPublished {
		return nil, nil, apperr.Validation("progress can only be recorded on published lessons")
	}
	if err := s.requirePrerequisite(ctx, actorID, lesson); err != nil {
		return nil, nil, err
	}

	existing, err := s.lessonProgress.GetByUserLesson(ctx, actorID, lessonID)
	if err != nil && !apperr.IsNotFound(err) {
		return nil, nil, err
	}

	now := time.Now().UTC()
	p := &domain.LessonProgress{
		UserID:   actorID,
		CourseID: courseID,
		LessonID: lessonID,
		Status:   domain.LessonProgressInProgress,
		Percent:  0,
	}
	if existing != nil {
		p = existing
	}

	if in.LastPosition != nil {
		p.LastPosition = strings.TrimSpace(*in.LastPosition)
	}
	if in.Percent != nil {
		if *in.Percent < 0 || *in.Percent > 100 {
			return nil, nil, apperr.Validation("percent must be between 0 and 100")
		}
		// Idempotent: percent only moves forward.
		if *in.Percent > p.Percent {
			p.Percent = *in.Percent
		}
	}

	complete := in.Completed != nil && *in.Completed
	if complete || p.Percent >= 100 {
		p.Status = domain.LessonProgressCompleted
		p.Percent = 100
		if p.CompletedAt == nil {
			p.CompletedAt = &now
		}
	} else if p.Status != domain.LessonProgressCompleted {
		p.Status = domain.LessonProgressInProgress
		p.CompletedAt = nil
	}

	if err := s.lessonProgress.Upsert(ctx, p); err != nil {
		return nil, nil, err
	}
	saved, err := s.lessonProgress.GetByUserLesson(ctx, actorID, lessonID)
	if err != nil {
		return nil, nil, err
	}

	courseProg, err := s.recomputeCourseProgress(ctx, actorID, courseID, enrollment.ID, &lessonID)
	if err != nil {
		return nil, nil, err
	}
	return saved, courseProg, nil
}

func (s *ProgressService) GetLessonProgress(ctx context.Context, actorID, lessonID string) (*domain.LessonProgress, error) {
	_, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, courseID); err != nil {
		return nil, err
	}
	p, err := s.lessonProgress.GetByUserLesson(ctx, actorID, lessonID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return &domain.LessonProgress{
				UserID:   actorID,
				CourseID: courseID,
				LessonID: lessonID,
				Status:   domain.LessonProgressInProgress,
				Percent:  0,
			}, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *ProgressService) GetCourseProgress(ctx context.Context, actorID, courseID string) (*domain.CourseProgress, []domain.LessonProgress, error) {
	if _, err := s.requireActiveEnrollment(ctx, actorID, courseID); err != nil {
		return nil, nil, err
	}
	cp, err := s.courseProgress.GetByUserCourse(ctx, actorID, courseID)
	if err != nil {
		if !apperr.IsNotFound(err) {
			return nil, nil, err
		}
		en, err := s.enrollments.GetByCourseUser(ctx, courseID, actorID)
		if err != nil {
			return nil, nil, err
		}
		cp, err = s.recomputeCourseProgress(ctx, actorID, courseID, en.ID, nil)
		if err != nil {
			return nil, nil, err
		}
	}
	lessons, err := s.lessonProgress.ListByUserCourse(ctx, actorID, courseID)
	if err != nil {
		return nil, nil, err
	}
	return cp, lessons, nil
}

func (s *ProgressService) ListMyProgress(ctx context.Context, actorID string) ([]domain.CourseProgressSummary, error) {
	items, err := s.courseProgress.ListByUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CourseProgressSummary, 0, len(items))
	for i := range items {
		sum := domain.CourseProgressSummary{CourseProgress: items[i]}
		if en, err := s.enrollments.GetByCourseUser(ctx, items[i].CourseID, actorID); err == nil {
			sum.EnrollmentStatus = en.Status
		}
		out = append(out, sum)
	}
	return out, nil
}

func (s *ProgressService) recomputeCourseProgress(ctx context.Context, userID, courseID, enrollmentID string, lastLessonID *string) (*domain.CourseProgress, error) {
	total, err := s.lessons.CountPublishedByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	completed, err := s.lessonProgress.CountCompletedPublished(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}
	percent := 0
	if total > 0 {
		percent = (completed * 100) / total
		if percent > 100 {
			percent = 100
		}
	}

	now := time.Now().UTC()
	cp := &domain.CourseProgress{
		UserID:           userID,
		CourseID:         courseID,
		EnrollmentID:     &enrollmentID,
		Percent:          percent,
		CompletedLessons: completed,
		TotalLessons:     total,
		LastLessonID:     lastLessonID,
	}
	if existing, err := s.courseProgress.GetByUserCourse(ctx, userID, courseID); err == nil {
		if lastLessonID == nil {
			cp.LastLessonID = existing.LastLessonID
		}
		cp.CompletedAt = existing.CompletedAt
	} else if !apperr.IsNotFound(err) {
		return nil, err
	}

	if total > 0 && completed >= total {
		if cp.CompletedAt == nil {
			cp.CompletedAt = &now
		}
		cp.Percent = 100
	} else {
		cp.CompletedAt = nil
	}

	if err := s.courseProgress.Upsert(ctx, cp); err != nil {
		return nil, err
	}
	return s.courseProgress.GetByUserCourse(ctx, userID, courseID)
}

func (s *ProgressService) requireActiveEnrollment(ctx context.Context, userID, courseID string) (*domain.Enrollment, error) {
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

func (s *ProgressService) requirePrerequisite(ctx context.Context, userID string, lesson *domain.Lesson) error {
	if lesson.PrerequisiteLessonID == nil || *lesson.PrerequisiteLessonID == "" {
		return nil
	}
	prereq, err := s.lessonProgress.GetByUserLesson(ctx, userID, *lesson.PrerequisiteLessonID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return apperr.Forbidden("complete the prerequisite lesson first")
		}
		return err
	}
	if prereq.Status != domain.LessonProgressCompleted {
		return apperr.Forbidden("complete the prerequisite lesson first")
	}
	return nil
}

func (s *ProgressService) lessonCourse(ctx context.Context, lessonID string) (*domain.Lesson, string, error) {
	lesson, err := s.lessons.GetByID(ctx, lessonID)
	if err != nil {
		return nil, "", err
	}
	mod, err := s.modules.GetByID(ctx, lesson.ModuleID)
	if err != nil {
		return nil, "", err
	}
	return lesson, mod.CourseID, nil
}
