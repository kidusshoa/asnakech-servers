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

type AssessmentService struct {
	courses     repository.CourseRepository
	enrollments repository.EnrollmentRepository
	quizzes     repository.QuizRepository
	questions   repository.QuizQuestionRepository
	attempts    repository.QuizAttemptRepository
	answers     repository.QuizAttemptAnswerRepository
	assignments repository.AssignmentRepository
	submissions repository.AssignmentSubmissionRepository
}

func NewAssessmentService(
	courses repository.CourseRepository,
	enrollments repository.EnrollmentRepository,
	quizzes repository.QuizRepository,
	questions repository.QuizQuestionRepository,
	attempts repository.QuizAttemptRepository,
	answers repository.QuizAttemptAnswerRepository,
	assignments repository.AssignmentRepository,
	submissions repository.AssignmentSubmissionRepository,
) *AssessmentService {
	return &AssessmentService{
		courses:     courses,
		enrollments: enrollments,
		quizzes:     quizzes,
		questions:   questions,
		attempts:    attempts,
		answers:     answers,
		assignments: assignments,
		submissions: submissions,
	}
}

// --- Quizzes (teacher) ---

func (s *AssessmentService) CreateQuiz(ctx context.Context, actorID, courseID, title, description string, timeLimit, maxAttempts *int, passPercent int, shuffle bool, platformAdmin bool) (*domain.Quiz, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	if passPercent < 0 || passPercent > 100 {
		passPercent = 60
	}
	q := &domain.Quiz{
		CourseID:         courseID,
		Title:            title,
		Description:      strings.TrimSpace(description),
		Status:           domain.QuizStatusDraft,
		TimeLimitSeconds: timeLimit,
		MaxAttempts:      maxAttempts,
		PassPercent:      passPercent,
		ShuffleQuestions: shuffle,
	}
	if err := s.quizzes.Create(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *AssessmentService) ListQuizzes(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.Quiz, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	items, err := s.quizzes.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if canAuthor {
		return items, nil
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, courseID); err != nil {
		return nil, err
	}
	out := make([]domain.Quiz, 0, len(items))
	for _, q := range items {
		if q.Status == domain.QuizStatusPublished {
			out = append(out, q)
		}
	}
	return out, nil
}

func (s *AssessmentService) GetQuiz(ctx context.Context, actorID, quizID string, platformAdmin bool) (*domain.Quiz, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, quiz.CourseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	if !canAuthor {
		if quiz.Status != domain.QuizStatusPublished {
			return nil, apperr.NotFound("quiz not found")
		}
		if _, err := s.requireActiveEnrollment(ctx, actorID, quiz.CourseID); err != nil {
			return nil, err
		}
	}
	questions, err := s.questions.ListByQuiz(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if !canAuthor {
		for i := range questions {
			stripQuestionSecrets(&questions[i])
		}
	}
	quiz.Questions = questions
	return quiz, nil
}

func (s *AssessmentService) UpdateQuiz(ctx context.Context, actorID, quizID, title, description string, timeLimit, maxAttempts *int, passPercent *int, shuffle *bool, platformAdmin bool) (*domain.Quiz, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, quiz.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if title = strings.TrimSpace(title); title != "" {
		quiz.Title = title
	}
	quiz.Description = strings.TrimSpace(description)
	quiz.TimeLimitSeconds = timeLimit
	quiz.MaxAttempts = maxAttempts
	if passPercent != nil {
		if *passPercent < 0 || *passPercent > 100 {
			return nil, apperr.Validation("pass_percent must be 0-100")
		}
		quiz.PassPercent = *passPercent
	}
	if shuffle != nil {
		quiz.ShuffleQuestions = *shuffle
	}
	return s.quizzes.Update(ctx, quiz)
}

func (s *AssessmentService) PublishQuiz(ctx context.Context, actorID, quizID string, platformAdmin bool) (*domain.Quiz, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, quiz.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	qs, err := s.questions.ListByQuiz(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if len(qs) == 0 {
		return nil, apperr.Validation("quiz must have at least one question")
	}
	return s.quizzes.SetStatus(ctx, quizID, domain.QuizStatusPublished)
}

func (s *AssessmentService) AddQuestion(ctx context.Context, actorID, quizID string, qType domain.QuestionType, prompt string, points int, options []domain.QuizOption, correctAnswer string, platformAdmin bool) (*domain.QuizQuestion, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, quiz.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, apperr.Validation("prompt is required")
	}
	if points <= 0 {
		points = 1
	}
	if err := validateQuestion(qType, options, correctAnswer); err != nil {
		return nil, err
	}
	ensureOptionIDs(options)
	pos, err := s.questions.NextPosition(ctx, quizID)
	if err != nil {
		return nil, err
	}
	q := &domain.QuizQuestion{
		QuizID:        quizID,
		QuestionType:  qType,
		Prompt:        prompt,
		Points:        points,
		Position:      pos,
		Options:       options,
		CorrectAnswer: strings.TrimSpace(correctAnswer),
	}
	if err := s.questions.Create(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *AssessmentService) UpdateQuestion(ctx context.Context, actorID, questionID string, prompt string, points int, options []domain.QuizOption, correctAnswer string, platformAdmin bool) (*domain.QuizQuestion, error) {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	quiz, err := s.quizzes.GetByID(ctx, q.QuizID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, quiz.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, apperr.Validation("prompt is required")
	}
	if points <= 0 {
		return nil, apperr.Validation("points must be > 0")
	}
	if err := validateQuestion(q.QuestionType, options, correctAnswer); err != nil {
		return nil, err
	}
	ensureOptionIDs(options)
	q.Prompt = prompt
	q.Points = points
	q.Options = options
	q.CorrectAnswer = strings.TrimSpace(correctAnswer)
	return s.questions.Update(ctx, q)
}

func (s *AssessmentService) DeleteQuestion(ctx context.Context, actorID, questionID string, platformAdmin bool) error {
	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	quiz, err := s.quizzes.GetByID(ctx, q.QuizID)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, quiz.CourseID, platformAdmin); err != nil {
		return err
	}
	return s.questions.Delete(ctx, questionID)
}

func (s *AssessmentService) ReorderQuestions(ctx context.Context, actorID, quizID string, ids []string, platformAdmin bool) error {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, quiz.CourseID, platformAdmin); err != nil {
		return err
	}
	return s.questions.Reorder(ctx, quizID, ids)
}

// --- Quiz attempts (student) ---

func (s *AssessmentService) StartAttempt(ctx context.Context, actorID, quizID string) (*domain.QuizAttempt, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.Status != domain.QuizStatusPublished {
		return nil, apperr.Validation("quiz is not published")
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, quiz.CourseID); err != nil {
		return nil, err
	}
	if existing, err := s.attempts.GetInProgress(ctx, quizID, actorID); err == nil {
		return existing, nil
	} else if !apperr.IsNotFound(err) {
		return nil, err
	}
	count, err := s.attempts.CountByQuizUser(ctx, quizID, actorID)
	if err != nil {
		return nil, err
	}
	if quiz.MaxAttempts != nil && count >= *quiz.MaxAttempts {
		return nil, apperr.Conflict("maximum attempts reached")
	}
	a := &domain.QuizAttempt{
		QuizID:        quizID,
		UserID:        actorID,
		AttemptNumber: count + 1,
		Status:        domain.AttemptStatusInProgress,
		StartedAt:     time.Now().UTC(),
	}
	if err := s.attempts.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AssessmentService) ListMyAttempts(ctx context.Context, actorID, quizID string) ([]domain.QuizAttempt, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, quiz.CourseID); err != nil {
		return nil, err
	}
	return s.attempts.ListByQuizUser(ctx, quizID, actorID)
}

func (s *AssessmentService) GetAttempt(ctx context.Context, actorID, attemptID string, platformAdmin bool) (*domain.QuizAttempt, error) {
	attempt, err := s.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	quiz, err := s.quizzes.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, quiz.CourseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	if !canAuthor && attempt.UserID != actorID {
		return nil, apperr.Forbidden("cannot view this attempt")
	}
	answers, err := s.answers.ListByAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	attempt.Answers = answers
	return attempt, nil
}

func (s *AssessmentService) SaveAnswers(ctx context.Context, actorID, attemptID string, answers []domain.QuizAttemptAnswer) (*domain.QuizAttempt, error) {
	attempt, err := s.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.UserID != actorID {
		return nil, apperr.Forbidden("cannot modify this attempt")
	}
	if attempt.Status != domain.AttemptStatusInProgress {
		return nil, apperr.Conflict("attempt is already submitted")
	}
	quiz, err := s.quizzes.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	if quiz.TimeLimitSeconds != nil {
		deadline := attempt.StartedAt.Add(time.Duration(*quiz.TimeLimitSeconds) * time.Second)
		if time.Now().UTC().After(deadline) {
			return nil, apperr.Conflict("time limit exceeded")
		}
	}
	questions, err := s.questions.ListByQuiz(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	qset := map[string]struct{}{}
	for _, q := range questions {
		qset[q.ID] = struct{}{}
	}
	clean := make([]domain.QuizAttemptAnswer, 0, len(answers))
	for _, a := range answers {
		if _, ok := qset[a.QuestionID]; !ok {
			return nil, apperr.Validation("invalid question_id")
		}
		a.AttemptID = attemptID
		if a.SelectedOptionIDs == nil {
			a.SelectedOptionIDs = []string{}
		}
		a.TextAnswer = strings.TrimSpace(a.TextAnswer)
		a.IsCorrect = nil
		a.PointsAwarded = 0
		clean = append(clean, a)
	}
	if err := s.answers.ReplaceAll(ctx, attemptID, clean); err != nil {
		return nil, err
	}
	return s.GetAttempt(ctx, actorID, attemptID, false)
}

func (s *AssessmentService) SubmitAttempt(ctx context.Context, actorID, attemptID string) (*domain.QuizAttempt, error) {
	attempt, err := s.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.UserID != actorID {
		return nil, apperr.Forbidden("cannot submit this attempt")
	}
	if attempt.Status != domain.AttemptStatusInProgress {
		return nil, apperr.Conflict("attempt is already submitted")
	}
	quiz, err := s.quizzes.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	questions, err := s.questions.ListByQuiz(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	existing, err := s.answers.ListByAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	byQ := map[string]domain.QuizAttemptAnswer{}
	for _, a := range existing {
		byQ[a.QuestionID] = a
	}

	now := time.Now().UTC()
	score, maxPts := 0, 0
	graded := make([]domain.QuizAttemptAnswer, 0, len(questions))
	for _, q := range questions {
		maxPts += q.Points
		ans, ok := byQ[q.ID]
		if !ok {
			ans = domain.QuizAttemptAnswer{AttemptID: attemptID, QuestionID: q.ID, SelectedOptionIDs: []string{}}
		}
		correct, pts := gradeAnswer(q, ans)
		ans.IsCorrect = &correct
		ans.PointsAwarded = pts
		score += pts
		graded = append(graded, ans)
	}
	if err := s.answers.ReplaceAll(ctx, attemptID, graded); err != nil {
		return nil, err
	}
	percent := 0
	if maxPts > 0 {
		percent = (score * 100) / maxPts
	}
	attempt.Status = domain.AttemptStatusGraded
	attempt.ScorePoints = score
	attempt.MaxPoints = maxPts
	attempt.Percent = percent
	attempt.Passed = percent >= quiz.PassPercent
	attempt.SubmittedAt = &now
	attempt.GradedAt = &now
	if err := s.attempts.Update(ctx, attempt); err != nil {
		return nil, err
	}
	return s.GetAttempt(ctx, actorID, attemptID, false)
}

// --- Assignments ---

func (s *AssessmentService) CreateAssignment(ctx context.Context, actorID, courseID, title, description string, maxScore int, dueAt *time.Time, allowLate bool, rubric []domain.RubricCriterion, platformAdmin bool) (*domain.Assignment, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	if maxScore <= 0 {
		maxScore = 100
	}
	ensureRubricIDs(rubric)
	a := &domain.Assignment{
		CourseID:    courseID,
		Title:       title,
		Description: strings.TrimSpace(description),
		Status:      domain.AssignmentStatusDraft,
		MaxScore:    maxScore,
		DueAt:       dueAt,
		AllowLate:   allowLate,
		Rubric:      rubric,
	}
	if err := s.assignments.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AssessmentService) ListAssignments(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.Assignment, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	items, err := s.assignments.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if canAuthor {
		return items, nil
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, courseID); err != nil {
		return nil, err
	}
	out := make([]domain.Assignment, 0, len(items))
	for _, a := range items {
		if a.Status == domain.AssignmentStatusPublished {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s *AssessmentService) GetAssignment(ctx context.Context, actorID, assignmentID string, platformAdmin bool) (*domain.Assignment, error) {
	a, err := s.assignments.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, a.CourseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || course.TeacherID == actorID
	if !canAuthor {
		if a.Status != domain.AssignmentStatusPublished {
			return nil, apperr.NotFound("assignment not found")
		}
		if _, err := s.requireActiveEnrollment(ctx, actorID, a.CourseID); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (s *AssessmentService) UpdateAssignment(ctx context.Context, actorID, assignmentID, title, description string, maxScore int, dueAt *time.Time, allowLate bool, rubric []domain.RubricCriterion, platformAdmin bool) (*domain.Assignment, error) {
	a, err := s.assignments.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, a.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if title = strings.TrimSpace(title); title != "" {
		a.Title = title
	}
	a.Description = strings.TrimSpace(description)
	if maxScore > 0 {
		a.MaxScore = maxScore
	}
	a.DueAt = dueAt
	a.AllowLate = allowLate
	ensureRubricIDs(rubric)
	a.Rubric = rubric
	return s.assignments.Update(ctx, a)
}

func (s *AssessmentService) PublishAssignment(ctx context.Context, actorID, assignmentID string, platformAdmin bool) (*domain.Assignment, error) {
	a, err := s.assignments.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, a.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.assignments.SetStatus(ctx, assignmentID, domain.AssignmentStatusPublished)
}

func (s *AssessmentService) UpsertSubmission(ctx context.Context, actorID, assignmentID, body, attachmentURL string, submit bool) (*domain.AssignmentSubmission, error) {
	a, err := s.assignments.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if a.Status != domain.AssignmentStatusPublished {
		return nil, apperr.Validation("assignment is not published")
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, a.CourseID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if submit && a.DueAt != nil && now.After(*a.DueAt) && !a.AllowLate {
		return nil, apperr.Forbidden("past due date")
	}

	existing, err := s.submissions.GetByAssignmentUser(ctx, assignmentID, actorID)
	if err != nil && !apperr.IsNotFound(err) {
		return nil, err
	}
	sub := &domain.AssignmentSubmission{
		AssignmentID:  assignmentID,
		UserID:        actorID,
		Status:        domain.SubmissionStatusDraft,
		Body:          strings.TrimSpace(body),
		AttachmentURL: strings.TrimSpace(attachmentURL),
		RubricScores:  map[string]int{},
	}
	if existing != nil {
		if existing.Status == domain.SubmissionStatusGraded || existing.Status == domain.SubmissionStatusReturned {
			return nil, apperr.Conflict("submission already graded")
		}
		sub = existing
		sub.Body = strings.TrimSpace(body)
		sub.AttachmentURL = strings.TrimSpace(attachmentURL)
	}
	if submit {
		sub.Status = domain.SubmissionStatusSubmitted
		sub.SubmittedAt = &now
	} else {
		sub.Status = domain.SubmissionStatusDraft
	}
	if err := s.submissions.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return s.submissions.GetByAssignmentUser(ctx, assignmentID, actorID)
}

func (s *AssessmentService) GetMySubmission(ctx context.Context, actorID, assignmentID string) (*domain.AssignmentSubmission, error) {
	a, err := s.assignments.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireActiveEnrollment(ctx, actorID, a.CourseID); err != nil {
		return nil, err
	}
	sub, err := s.submissions.GetByAssignmentUser(ctx, assignmentID, actorID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return &domain.AssignmentSubmission{
				AssignmentID: assignmentID,
				UserID:       actorID,
				Status:       domain.SubmissionStatusDraft,
				RubricScores: map[string]int{},
			}, nil
		}
		return nil, err
	}
	return sub, nil
}

func (s *AssessmentService) ListSubmissions(ctx context.Context, actorID, assignmentID string, platformAdmin bool) ([]domain.AssignmentSubmission, error) {
	a, err := s.assignments.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, a.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.submissions.ListByAssignment(ctx, assignmentID)
}

func (s *AssessmentService) GradeSubmission(ctx context.Context, actorID, submissionID string, score int, feedback string, rubricScores map[string]int, platformAdmin bool) (*domain.AssignmentSubmission, error) {
	sub, err := s.submissions.GetByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	a, err := s.assignments.GetByID(ctx, sub.AssignmentID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, a.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	if sub.Status != domain.SubmissionStatusSubmitted && sub.Status != domain.SubmissionStatusGraded && sub.Status != domain.SubmissionStatusReturned {
		return nil, apperr.Validation("submission is not ready for grading")
	}
	if score < 0 || score > a.MaxScore {
		return nil, apperr.Validation("score must be between 0 and max_score")
	}
	if rubricScores == nil {
		rubricScores = map[string]int{}
	}
	now := time.Now().UTC()
	sub.Score = &score
	sub.Feedback = strings.TrimSpace(feedback)
	sub.RubricScores = rubricScores
	sub.Status = domain.SubmissionStatusGraded
	sub.GradedAt = &now
	sub.GradedBy = &actorID
	if err := s.submissions.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return s.submissions.GetByID(ctx, submissionID)
}

func (s *AssessmentService) Gradebook(ctx context.Context, actorID, courseID string, platformAdmin bool) ([]domain.GradebookEntry, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	enrollments, _, err := s.enrollments.ListByCourse(ctx, courseID, domain.EnrollmentListFilter{
		Page: 1, PerPage: 100, Status: domain.EnrollmentStatusActive,
	})
	if err != nil {
		return nil, err
	}
	quizzes, err := s.quizzes.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	assignments, err := s.assignments.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	bestAttempts, err := s.attempts.BestGradedByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	subs, err := s.submissions.ListGradedByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	type key struct{ user, quiz string }
	best := map[key]domain.QuizAttempt{}
	attemptCounts := map[key]int{}
	for _, a := range bestAttempts {
		k := key{a.UserID, a.QuizID}
		best[k] = a
	}
	for _, q := range quizzes {
		for _, en := range enrollments {
			n, _ := s.attempts.CountByQuizUser(ctx, q.ID, en.UserID)
			attemptCounts[key{en.UserID, q.ID}] = n
		}
	}

	subByUserAssign := map[string]map[string]domain.AssignmentSubmission{}
	for _, sub := range subs {
		if subByUserAssign[sub.UserID] == nil {
			subByUserAssign[sub.UserID] = map[string]domain.AssignmentSubmission{}
		}
		subByUserAssign[sub.UserID][sub.AssignmentID] = sub
	}

	out := make([]domain.GradebookEntry, 0, len(enrollments))
	for _, en := range enrollments {
		entry := domain.GradebookEntry{
			UserID:       en.UserID,
			UserEmail:    en.UserEmail,
			UserFullName: en.UserFullName,
			Quizzes:      make([]domain.GradebookQuizScore, 0, len(quizzes)),
			Assignments:  make([]domain.GradebookAssignmentScore, 0, len(assignments)),
		}
		for _, q := range quizzes {
			gs := domain.GradebookQuizScore{QuizID: q.ID, QuizTitle: q.Title, Attempts: attemptCounts[key{en.UserID, q.ID}]}
			if ba, ok := best[key{en.UserID, q.ID}]; ok {
				p, passed := ba.Percent, ba.Passed
				gs.Percent = &p
				gs.Passed = &passed
			}
			entry.Quizzes = append(entry.Quizzes, gs)
		}
		for _, a := range assignments {
			as := domain.GradebookAssignmentScore{
				AssignmentID: a.ID, AssignmentTitle: a.Title, MaxScore: a.MaxScore, Status: "",
			}
			if m := subByUserAssign[en.UserID]; m != nil {
				if sub, ok := m[a.ID]; ok {
					as.Score = sub.Score
					as.Status = string(sub.Status)
				}
			}
			entry.Assignments = append(entry.Assignments, as)
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *AssessmentService) requireCourseWrite(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin || course.TeacherID == actorID {
		return nil
	}
	return apperr.Forbidden("only the course teacher or an admin can manage assessments")
}

func (s *AssessmentService) requireActiveEnrollment(ctx context.Context, userID, courseID string) (*domain.Enrollment, error) {
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

func validateQuestion(qType domain.QuestionType, options []domain.QuizOption, correctAnswer string) error {
	switch qType {
	case domain.QuestionTypeMCQ:
		if len(options) < 2 {
			return apperr.Validation("mcq requires at least 2 options")
		}
		hasCorrect := false
		for _, o := range options {
			if strings.TrimSpace(o.Text) == "" {
				return apperr.Validation("option text is required")
			}
			if o.IsCorrect {
				hasCorrect = true
			}
		}
		if !hasCorrect {
			return apperr.Validation("mcq requires at least one correct option")
		}
	case domain.QuestionTypeShortAnswer:
		if strings.TrimSpace(correctAnswer) == "" {
			return apperr.Validation("short_answer requires correct_answer")
		}
	default:
		return apperr.Validation("invalid question_type")
	}
	return nil
}

func gradeAnswer(q domain.QuizQuestion, ans domain.QuizAttemptAnswer) (bool, int) {
	switch q.QuestionType {
	case domain.QuestionTypeMCQ:
		correctIDs := map[string]struct{}{}
		for _, o := range q.Options {
			if o.IsCorrect {
				correctIDs[o.ID] = struct{}{}
			}
		}
		if len(ans.SelectedOptionIDs) != len(correctIDs) {
			return false, 0
		}
		for _, id := range ans.SelectedOptionIDs {
			if _, ok := correctIDs[id]; !ok {
				return false, 0
			}
		}
		return true, q.Points
	case domain.QuestionTypeShortAnswer:
		if strings.EqualFold(strings.TrimSpace(ans.TextAnswer), strings.TrimSpace(q.CorrectAnswer)) {
			return true, q.Points
		}
		return false, 0
	default:
		return false, 0
	}
}

func stripQuestionSecrets(q *domain.QuizQuestion) {
	q.CorrectAnswer = ""
	for i := range q.Options {
		q.Options[i].IsCorrect = false
	}
}

func ensureOptionIDs(options []domain.QuizOption) {
	for i := range options {
		if strings.TrimSpace(options[i].ID) == "" {
			options[i].ID = randomID(4)
		}
	}
}

func ensureRubricIDs(rubric []domain.RubricCriterion) {
	for i := range rubric {
		if strings.TrimSpace(rubric[i].ID) == "" {
			rubric[i].ID = randomID(4)
		}
	}
}

func randomID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
