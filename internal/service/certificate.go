package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	certpdf "github.com/asnakech/asnakech-servers/internal/certificate"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type CertificateService struct {
	courses        repository.CourseRepository
	enrollments    repository.EnrollmentRepository
	users          repository.UserRepository
	courseProgress repository.CourseProgressRepository
	certificates   repository.CertificateRepository
	quizzes        repository.QuizRepository
	assignments    repository.AssignmentRepository
	attempts       repository.QuizAttemptRepository
	submissions    repository.AssignmentSubmissionRepository
}

func NewCertificateService(
	courses repository.CourseRepository,
	enrollments repository.EnrollmentRepository,
	users repository.UserRepository,
	courseProgress repository.CourseProgressRepository,
	certificates repository.CertificateRepository,
	quizzes repository.QuizRepository,
	assignments repository.AssignmentRepository,
	attempts repository.QuizAttemptRepository,
	submissions repository.AssignmentSubmissionRepository,
) *CertificateService {
	return &CertificateService{
		courses:        courses,
		enrollments:    enrollments,
		users:          users,
		courseProgress: courseProgress,
		certificates:   certificates,
		quizzes:        quizzes,
		assignments:    assignments,
		attempts:       attempts,
		submissions:    submissions,
	}
}

func (s *CertificateService) Issue(ctx context.Context, actorID, courseID string, targetUserID *string, platformAdmin bool) (*domain.Certificate, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	userID := actorID
	canOverride := platformAdmin || course.TeacherID == actorID
	if targetUserID != nil && strings.TrimSpace(*targetUserID) != "" {
		if !canOverride {
			return nil, apperr.Forbidden("only the course teacher or an admin can issue for another user")
		}
		userID = strings.TrimSpace(*targetUserID)
	} else if !canOverride {
		if _, err := s.requireActiveEnrollment(ctx, userID, courseID); err != nil {
			return nil, err
		}
		if err := s.requireCourseComplete(ctx, userID, courseID); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.enrollments.GetByCourseUser(ctx, courseID, userID); err != nil {
			if apperr.IsNotFound(err) {
				return nil, apperr.Validation("user is not enrolled in this course")
			}
			return nil, err
		}
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	code, err := newVerificationCode()
	if err != nil {
		return nil, apperr.Internal("failed to generate verification code")
	}

	learnerName := strings.TrimSpace(user.FullName)
	if learnerName == "" {
		learnerName = user.Email
	}

	cert := &domain.Certificate{
		CourseID:         courseID,
		UserID:           userID,
		VerificationCode: code,
		LearnerName:      learnerName,
		CourseTitle:      course.Title,
		IssuedAt:         now,
	}
	if err := s.certificates.Create(ctx, cert); err != nil {
		return nil, err
	}
	return s.certificates.GetByID(ctx, cert.ID)
}

func (s *CertificateService) ListMine(ctx context.Context, actorID string) ([]domain.Certificate, error) {
	return s.certificates.ListByUser(ctx, actorID)
}

func (s *CertificateService) ListCourseCertificates(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.Certificate, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.certificates.ListByCourse(ctx, courseID)
}

func (s *CertificateService) Get(ctx context.Context, actorID, certID string, platformAdmin bool) (*domain.Certificate, error) {
	cert, err := s.certificates.GetByID(ctx, certID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCertificateRead(ctx, actorID, cert, platformAdmin); err != nil {
		return nil, err
	}
	return cert, nil
}

func (s *CertificateService) Verify(ctx context.Context, code string) (*domain.CertificateVerify, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, apperr.Validation("verification code is required")
	}
	cert, err := s.certificates.GetByVerificationCode(ctx, code)
	if err != nil {
		if apperr.IsNotFound(err) {
			return &domain.CertificateVerify{Valid: false, VerificationCode: code}, nil
		}
		return nil, err
	}
	valid := cert.RevokedAt == nil
	return &domain.CertificateVerify{
		Valid:            valid,
		VerificationCode: cert.VerificationCode,
		LearnerName:      cert.LearnerName,
		CourseTitle:      cert.CourseTitle,
		IssuedAt:         cert.IssuedAt,
		RevokedAt:        cert.RevokedAt,
	}, nil
}

func (s *CertificateService) DownloadPDF(ctx context.Context, actorID, certID string, platformAdmin bool) ([]byte, string, error) {
	cert, err := s.Get(ctx, actorID, certID, platformAdmin)
	if err != nil {
		return nil, "", err
	}
	if cert.RevokedAt != nil {
		return nil, "", apperr.Forbidden("certificate has been revoked")
	}
	pdf, err := certpdf.GeneratePDF(certpdf.PDFData{
		LearnerName:      cert.LearnerName,
		CourseTitle:      cert.CourseTitle,
		IssuedAt:         cert.IssuedAt,
		VerificationCode: cert.VerificationCode,
	})
	if err != nil {
		return nil, "", err
	}
	filename := "certificate-" + cert.VerificationCode + ".pdf"
	return pdf, filename, nil
}

func (s *CertificateService) Revoke(ctx context.Context, actorID, certID string, platformAdmin bool) (*domain.Certificate, error) {
	cert, err := s.certificates.GetByID(ctx, certID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, cert.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.certificates.Revoke(ctx, certID)
}

func (s *CertificateService) MyTranscript(ctx context.Context, actorID string) (*domain.Transcript, error) {
	user, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return s.buildTranscript(ctx, user)
}

func (s *CertificateService) UserTranscript(ctx context.Context, actorID, courseID, targetUserID string, platformAdmin bool) (*domain.Transcript, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	if _, err := s.enrollments.GetByCourseUser(ctx, courseID, targetUserID); err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	tr, err := s.buildTranscript(ctx, user)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.TranscriptCourse, 0, 1)
	for _, c := range tr.Courses {
		if c.CourseID == courseID {
			filtered = append(filtered, c)
		}
	}
	tr.Courses = filtered
	return tr, nil
}

func (s *CertificateService) buildTranscript(ctx context.Context, user *domain.User) (*domain.Transcript, error) {
	enrollments, _, err := s.enrollments.ListByUser(ctx, user.ID, domain.EnrollmentListFilter{
		Page: 1, PerPage: 500, Status: domain.EnrollmentStatusActive,
	})
	if err != nil {
		return nil, err
	}

	tr := &domain.Transcript{
		UserID:       user.ID,
		UserEmail:    user.Email,
		UserFullName: user.FullName,
		GeneratedAt:  time.Now().UTC(),
		Courses:      make([]domain.TranscriptCourse, 0, len(enrollments)),
	}

	for _, en := range enrollments {
		entry, err := s.transcriptCourse(ctx, user.ID, en.CourseID, en.CourseTitle, en.CourseSlug)
		if err != nil {
			return nil, err
		}
		tr.Courses = append(tr.Courses, *entry)
	}
	return tr, nil
}

func (s *CertificateService) transcriptCourse(ctx context.Context, userID, courseID, title, slug string) (*domain.TranscriptCourse, error) {
	entry := &domain.TranscriptCourse{
		CourseID:    courseID,
		CourseTitle: title,
		CourseSlug:  slug,
	}

	if cp, err := s.courseProgress.GetByUserCourse(ctx, userID, courseID); err == nil {
		entry.ProgressPercent = cp.Percent
		entry.CompletedAt = cp.CompletedAt
	}

	quizzes, _ := s.quizzes.ListByCourse(ctx, courseID)
	assignments, _ := s.assignments.ListByCourse(ctx, courseID)

	for _, q := range quizzes {
		if q.Status != domain.QuizStatusPublished {
			continue
		}
		gs := domain.GradebookQuizScore{QuizID: q.ID, QuizTitle: q.Title}
		if n, err := s.attempts.CountByQuizUser(ctx, q.ID, userID); err == nil {
			gs.Attempts = n
		}
		if attempts, err := s.attempts.ListByQuizUser(ctx, q.ID, userID); err == nil {
			for _, a := range attempts {
				if a.Status == domain.AttemptStatusGraded && (gs.Percent == nil || a.Percent > *gs.Percent) {
					p, passed := a.Percent, a.Passed
					gs.Percent = &p
					gs.Passed = &passed
				}
			}
		}
		entry.Quizzes = append(entry.Quizzes, gs)
	}

	for _, a := range assignments {
		if a.Status != domain.AssignmentStatusPublished {
			continue
		}
		as := domain.GradebookAssignmentScore{
			AssignmentID: a.ID, AssignmentTitle: a.Title, MaxScore: a.MaxScore,
		}
		if sub, err := s.submissions.GetByAssignmentUser(ctx, a.ID, userID); err == nil {
			as.Score = sub.Score
			as.Status = string(sub.Status)
		}
		entry.Assignments = append(entry.Assignments, as)
	}

	if cert, err := s.certificates.GetByCourseUser(ctx, courseID, userID); err == nil {
		entry.Certificate = &domain.TranscriptCertificate{
			ID:               cert.ID,
			VerificationCode: cert.VerificationCode,
			IssuedAt:         cert.IssuedAt,
			Revoked:          cert.RevokedAt != nil,
		}
	}

	return entry, nil
}

func (s *CertificateService) requireCourseComplete(ctx context.Context, userID, courseID string) error {
	cp, err := s.courseProgress.GetByUserCourse(ctx, userID, courseID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return apperr.Validation("course is not complete")
		}
		return err
	}
	if cp.Percent < 100 || cp.CompletedAt == nil {
		return apperr.Validation("course must be 100% complete before issuing a certificate")
	}
	return nil
}

func (s *CertificateService) requireCourseWrite(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	return apperr.Forbidden("only the course teacher or an admin can manage certificates")
}

func (s *CertificateService) requireActiveEnrollment(ctx context.Context, userID, courseID string) (*domain.Enrollment, error) {
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

func (s *CertificateService) requireCertificateRead(ctx context.Context, actorID string, cert *domain.Certificate, platformAdmin bool) error {
	if cert.UserID == actorID {
		return nil
	}
	return s.requireCourseWrite(ctx, actorID, cert.CourseID, platformAdmin)
}

func newVerificationCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}
