package handlers

import (
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type AssessmentHandler struct {
	assessments *service.AssessmentService
}

func NewAssessmentHandler(assessments *service.AssessmentService) *AssessmentHandler {
	return &AssessmentHandler{assessments: assessments}
}

type createQuizRequest struct {
	Title            string `json:"title" binding:"required"`
	Description      string `json:"description"`
	TimeLimitSeconds *int   `json:"time_limit_seconds"`
	MaxAttempts      *int   `json:"max_attempts"`
	PassPercent      int    `json:"pass_percent"`
	ShuffleQuestions bool   `json:"shuffle_questions"`
}

type updateQuizRequest struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	TimeLimitSeconds *int   `json:"time_limit_seconds"`
	MaxAttempts      *int   `json:"max_attempts"`
	PassPercent      *int   `json:"pass_percent"`
	ShuffleQuestions *bool  `json:"shuffle_questions"`
}

type quizOptionRequest struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type addQuestionRequest struct {
	QuestionType  string              `json:"question_type" binding:"required"`
	Prompt        string              `json:"prompt" binding:"required"`
	Points        int                 `json:"points"`
	Options       []quizOptionRequest `json:"options"`
	CorrectAnswer string              `json:"correct_answer"`
}

type updateQuestionRequest struct {
	Prompt        string              `json:"prompt" binding:"required"`
	Points        int                 `json:"points"`
	Options       []quizOptionRequest `json:"options"`
	CorrectAnswer string              `json:"correct_answer"`
}

type saveAnswersRequest struct {
	Answers []saveAnswerItem `json:"answers" binding:"required"`
}

type saveAnswerItem struct {
	QuestionID        string   `json:"question_id" binding:"required"`
	SelectedOptionIDs []string `json:"selected_option_ids"`
	TextAnswer        string   `json:"text_answer"`
}

type createAssignmentRequest struct {
	Title       string                   `json:"title" binding:"required"`
	Description string                   `json:"description"`
	MaxScore    int                      `json:"max_score"`
	DueAt       *time.Time               `json:"due_at"`
	AllowLate   bool                     `json:"allow_late"`
	Rubric      []rubricCriterionRequest `json:"rubric"`
}

type updateAssignmentRequest struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	MaxScore    int                      `json:"max_score"`
	DueAt       *time.Time               `json:"due_at"`
	AllowLate   bool                     `json:"allow_late"`
	Rubric      []rubricCriterionRequest `json:"rubric"`
}

type rubricCriterionRequest struct {
	ID        string `json:"id"`
	Criterion string `json:"criterion"`
	MaxPoints int    `json:"max_points"`
}

type upsertSubmissionRequest struct {
	Body          string `json:"body"`
	AttachmentURL string `json:"attachment_url"`
	Submit        bool   `json:"submit"`
}

type gradeSubmissionRequest struct {
	Score        int            `json:"score"`
	Feedback     string         `json:"feedback"`
	RubricScores map[string]int `json:"rubric_scores"`
}

type QuizOptionResponse struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct,omitempty"`
}

type QuestionResponse struct {
	ID            string               `json:"id"`
	QuizID        string               `json:"quiz_id"`
	QuestionType  string               `json:"question_type"`
	Prompt        string               `json:"prompt"`
	Points        int                  `json:"points"`
	Position      int                  `json:"position"`
	Options       []QuizOptionResponse `json:"options,omitempty"`
	CorrectAnswer string               `json:"correct_answer,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type QuizResponse struct {
	ID               string             `json:"id"`
	CourseID         string             `json:"course_id"`
	Title            string             `json:"title"`
	Description      string             `json:"description"`
	Status           string             `json:"status"`
	TimeLimitSeconds *int               `json:"time_limit_seconds,omitempty"`
	MaxAttempts      *int               `json:"max_attempts,omitempty"`
	PassPercent      int                `json:"pass_percent"`
	ShuffleQuestions bool               `json:"shuffle_questions"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	Questions        []QuestionResponse `json:"questions,omitempty"`
}

type AttemptAnswerResponse struct {
	ID                string    `json:"id,omitempty"`
	AttemptID         string    `json:"attempt_id,omitempty"`
	QuestionID        string    `json:"question_id"`
	SelectedOptionIDs []string  `json:"selected_option_ids"`
	TextAnswer        string    `json:"text_answer"`
	IsCorrect         *bool     `json:"is_correct,omitempty"`
	PointsAwarded     int       `json:"points_awarded"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

type AttemptResponse struct {
	ID            string                  `json:"id"`
	QuizID        string                  `json:"quiz_id"`
	UserID        string                  `json:"user_id"`
	AttemptNumber int                     `json:"attempt_number"`
	Status        string                  `json:"status"`
	ScorePoints   int                     `json:"score_points"`
	MaxPoints     int                     `json:"max_points"`
	Percent       int                     `json:"percent"`
	Passed        bool                    `json:"passed"`
	StartedAt     time.Time               `json:"started_at"`
	SubmittedAt   *time.Time              `json:"submitted_at,omitempty"`
	GradedAt      *time.Time              `json:"graded_at,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	Answers       []AttemptAnswerResponse `json:"answers,omitempty"`
	UserEmail     string                  `json:"user_email,omitempty"`
	UserFullName  string                  `json:"user_full_name,omitempty"`
}

type RubricCriterionResponse struct {
	ID        string `json:"id"`
	Criterion string `json:"criterion"`
	MaxPoints int    `json:"max_points"`
}

type AssignmentResponse struct {
	ID          string                    `json:"id"`
	CourseID    string                    `json:"course_id"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Status      string                    `json:"status"`
	MaxScore    int                       `json:"max_score"`
	DueAt       *time.Time                `json:"due_at,omitempty"`
	AllowLate   bool                      `json:"allow_late"`
	Rubric      []RubricCriterionResponse `json:"rubric,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

type SubmissionResponse struct {
	ID            string         `json:"id,omitempty"`
	AssignmentID  string         `json:"assignment_id"`
	UserID        string         `json:"user_id"`
	Status        string         `json:"status"`
	Body          string         `json:"body"`
	AttachmentURL string         `json:"attachment_url"`
	Score         *int           `json:"score,omitempty"`
	Feedback      string         `json:"feedback,omitempty"`
	RubricScores  map[string]int `json:"rubric_scores,omitempty"`
	SubmittedAt   *time.Time     `json:"submitted_at,omitempty"`
	GradedAt      *time.Time     `json:"graded_at,omitempty"`
	GradedBy      *string        `json:"graded_by,omitempty"`
	CreatedAt     time.Time      `json:"created_at,omitempty"`
	UpdatedAt     time.Time      `json:"updated_at,omitempty"`
	UserEmail     string         `json:"user_email,omitempty"`
	UserFullName  string         `json:"user_full_name,omitempty"`
}

type GradebookQuizScoreResponse struct {
	QuizID    string `json:"quiz_id"`
	QuizTitle string `json:"quiz_title"`
	Percent   *int   `json:"percent,omitempty"`
	Passed    *bool  `json:"passed,omitempty"`
	Attempts  int    `json:"attempts"`
}

type GradebookAssignmentScoreResponse struct {
	AssignmentID    string `json:"assignment_id"`
	AssignmentTitle string `json:"assignment_title"`
	Score           *int   `json:"score,omitempty"`
	MaxScore        int    `json:"max_score"`
	Status          string `json:"status,omitempty"`
}

type GradebookEntryResponse struct {
	UserID       string                             `json:"user_id"`
	UserEmail    string                             `json:"user_email"`
	UserFullName string                             `json:"user_full_name"`
	Quizzes      []GradebookQuizScoreResponse       `json:"quizzes"`
	Assignments  []GradebookAssignmentScoreResponse `json:"assignments"`
}

type QuizEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Data    QuizResponse `json:"data"`
}

type QuizListEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    []QuizResponse `json:"data"`
}

type QuestionEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    QuestionResponse `json:"data"`
}

type AttemptEnvelope struct {
	Success bool            `json:"success" example:"true"`
	Data    AttemptResponse `json:"data"`
}

type AttemptListEnvelope struct {
	Success bool              `json:"success" example:"true"`
	Data    []AttemptResponse `json:"data"`
}

type AssignmentEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    AssignmentResponse `json:"data"`
}

type AssignmentListEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []AssignmentResponse `json:"data"`
}

type SubmissionEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    SubmissionResponse `json:"data"`
}

type SubmissionListEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []SubmissionResponse `json:"data"`
}

type GradebookEnvelope struct {
	Success bool                     `json:"success" example:"true"`
	Data    []GradebookEntryResponse `json:"data"`
}

// CreateQuiz godoc
// @Summary      Create quiz
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body createQuizRequest true "Quiz"
// @Success      201 {object} QuizEnvelope
// @Router       /api/v1/courses/{id}/quizzes [post]
func (h *AssessmentHandler) CreateQuiz(c *gin.Context) {
	var req createQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	quiz, err := h.assessments.CreateQuiz(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.Title,
		req.Description,
		req.TimeLimitSeconds,
		req.MaxAttempts,
		req.PassPercent,
		req.ShuffleQuestions,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toQuizResponse(quiz))
}

// ListQuizzes godoc
// @Summary      List quizzes for a course
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} QuizListEnvelope
// @Router       /api/v1/courses/{id}/quizzes [get]
func (h *AssessmentHandler) ListQuizzes(c *gin.Context) {
	items, err := h.assessments.ListQuizzes(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]QuizResponse, 0, len(items))
	for i := range items {
		out = append(out, toQuizResponse(&items[i]))
	}
	response.OK(c, out)
}

// GetQuiz godoc
// @Summary      Get quiz with questions
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        quizId path string true "Quiz ID"
// @Success      200 {object} QuizEnvelope
// @Router       /api/v1/quizzes/{quizId} [get]
func (h *AssessmentHandler) GetQuiz(c *gin.Context) {
	quiz, err := h.assessments.GetQuiz(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("quizId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toQuizResponse(quiz))
}

// UpdateQuiz godoc
// @Summary      Update quiz
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        quizId path string true "Quiz ID"
// @Param        body body updateQuizRequest true "Quiz"
// @Success      200 {object} QuizEnvelope
// @Router       /api/v1/quizzes/{quizId} [patch]
func (h *AssessmentHandler) UpdateQuiz(c *gin.Context) {
	var req updateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	quiz, err := h.assessments.UpdateQuiz(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("quizId"),
		req.Title,
		req.Description,
		req.TimeLimitSeconds,
		req.MaxAttempts,
		req.PassPercent,
		req.ShuffleQuestions,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toQuizResponse(quiz))
}

// PublishQuiz godoc
// @Summary      Publish quiz
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        quizId path string true "Quiz ID"
// @Success      200 {object} QuizEnvelope
// @Router       /api/v1/quizzes/{quizId}/publish [post]
func (h *AssessmentHandler) PublishQuiz(c *gin.Context) {
	quiz, err := h.assessments.PublishQuiz(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("quizId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toQuizResponse(quiz))
}

// AddQuestion godoc
// @Summary      Add question to quiz
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        quizId path string true "Quiz ID"
// @Param        body body addQuestionRequest true "Question"
// @Success      201 {object} QuestionEnvelope
// @Router       /api/v1/quizzes/{quizId}/questions [post]
func (h *AssessmentHandler) AddQuestion(c *gin.Context) {
	var req addQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	q, err := h.assessments.AddQuestion(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("quizId"),
		domain.QuestionType(req.QuestionType),
		req.Prompt,
		req.Points,
		toDomainQuizOptions(req.Options),
		req.CorrectAnswer,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toQuestionResponse(q))
}

// UpdateQuestion godoc
// @Summary      Update quiz question
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        questionId path string true "Question ID"
// @Param        body body updateQuestionRequest true "Question"
// @Success      200 {object} QuestionEnvelope
// @Router       /api/v1/questions/{questionId} [patch]
func (h *AssessmentHandler) UpdateQuestion(c *gin.Context) {
	var req updateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	q, err := h.assessments.UpdateQuestion(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("questionId"),
		req.Prompt,
		req.Points,
		toDomainQuizOptions(req.Options),
		req.CorrectAnswer,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toQuestionResponse(q))
}

// DeleteQuestion godoc
// @Summary      Delete quiz question
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        questionId path string true "Question ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/questions/{questionId} [delete]
func (h *AssessmentHandler) DeleteQuestion(c *gin.Context) {
	if err := h.assessments.DeleteQuestion(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("questionId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "question deleted"})
}

// ReorderQuestions godoc
// @Summary      Reorder quiz questions
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        quizId path string true "Quiz ID"
// @Param        body body reorderRequest true "Ordered question IDs"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/quizzes/{quizId}/questions/reorder [put]
func (h *AssessmentHandler) ReorderQuestions(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	if err := h.assessments.ReorderQuestions(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("quizId"),
		req.IDs,
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "questions reordered"})
}

// StartAttempt godoc
// @Summary      Start quiz attempt
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        quizId path string true "Quiz ID"
// @Success      201 {object} AttemptEnvelope
// @Router       /api/v1/quizzes/{quizId}/attempts [post]
func (h *AssessmentHandler) StartAttempt(c *gin.Context) {
	attempt, err := h.assessments.StartAttempt(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("quizId"),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toAttemptResponse(attempt))
}

// ListMyAttempts godoc
// @Summary      List my quiz attempts
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        quizId path string true "Quiz ID"
// @Success      200 {object} AttemptListEnvelope
// @Router       /api/v1/quizzes/{quizId}/attempts [get]
func (h *AssessmentHandler) ListMyAttempts(c *gin.Context) {
	items, err := h.assessments.ListMyAttempts(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("quizId"),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]AttemptResponse, 0, len(items))
	for i := range items {
		out = append(out, toAttemptResponse(&items[i]))
	}
	response.OK(c, out)
}

// GetAttempt godoc
// @Summary      Get quiz attempt
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        attemptId path string true "Attempt ID"
// @Success      200 {object} AttemptEnvelope
// @Router       /api/v1/attempts/{attemptId} [get]
func (h *AssessmentHandler) GetAttempt(c *gin.Context) {
	attempt, err := h.assessments.GetAttempt(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("attemptId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAttemptResponse(attempt))
}

// SaveAnswers godoc
// @Summary      Save quiz attempt answers
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        attemptId path string true "Attempt ID"
// @Param        body body saveAnswersRequest true "Answers"
// @Success      200 {object} AttemptEnvelope
// @Router       /api/v1/attempts/{attemptId}/answers [put]
func (h *AssessmentHandler) SaveAnswers(c *gin.Context) {
	var req saveAnswersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	answers := make([]domain.QuizAttemptAnswer, 0, len(req.Answers))
	for _, a := range req.Answers {
		ids := a.SelectedOptionIDs
		if ids == nil {
			ids = []string{}
		}
		answers = append(answers, domain.QuizAttemptAnswer{
			QuestionID:        a.QuestionID,
			SelectedOptionIDs: ids,
			TextAnswer:        a.TextAnswer,
		})
	}
	attempt, err := h.assessments.SaveAnswers(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("attemptId"),
		answers,
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAttemptResponse(attempt))
}

// SubmitAttempt godoc
// @Summary      Submit quiz attempt
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        attemptId path string true "Attempt ID"
// @Success      200 {object} AttemptEnvelope
// @Router       /api/v1/attempts/{attemptId}/submit [post]
func (h *AssessmentHandler) SubmitAttempt(c *gin.Context) {
	attempt, err := h.assessments.SubmitAttempt(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("attemptId"),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAttemptResponse(attempt))
}

// CreateAssignment godoc
// @Summary      Create assignment
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body createAssignmentRequest true "Assignment"
// @Success      201 {object} AssignmentEnvelope
// @Router       /api/v1/courses/{id}/assignments [post]
func (h *AssessmentHandler) CreateAssignment(c *gin.Context) {
	var req createAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	a, err := h.assessments.CreateAssignment(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.Title,
		req.Description,
		req.MaxScore,
		req.DueAt,
		req.AllowLate,
		toDomainRubric(req.Rubric),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toAssignmentResponse(a))
}

// ListAssignments godoc
// @Summary      List assignments for a course
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} AssignmentListEnvelope
// @Router       /api/v1/courses/{id}/assignments [get]
func (h *AssessmentHandler) ListAssignments(c *gin.Context) {
	items, err := h.assessments.ListAssignments(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]AssignmentResponse, 0, len(items))
	for i := range items {
		out = append(out, toAssignmentResponse(&items[i]))
	}
	response.OK(c, out)
}

// GetAssignment godoc
// @Summary      Get assignment
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        assignmentId path string true "Assignment ID"
// @Success      200 {object} AssignmentEnvelope
// @Router       /api/v1/assignments/{assignmentId} [get]
func (h *AssessmentHandler) GetAssignment(c *gin.Context) {
	a, err := h.assessments.GetAssignment(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("assignmentId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAssignmentResponse(a))
}

// UpdateAssignment godoc
// @Summary      Update assignment
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        assignmentId path string true "Assignment ID"
// @Param        body body updateAssignmentRequest true "Assignment"
// @Success      200 {object} AssignmentEnvelope
// @Router       /api/v1/assignments/{assignmentId} [patch]
func (h *AssessmentHandler) UpdateAssignment(c *gin.Context) {
	var req updateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	a, err := h.assessments.UpdateAssignment(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("assignmentId"),
		req.Title,
		req.Description,
		req.MaxScore,
		req.DueAt,
		req.AllowLate,
		toDomainRubric(req.Rubric),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAssignmentResponse(a))
}

// PublishAssignment godoc
// @Summary      Publish assignment
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        assignmentId path string true "Assignment ID"
// @Success      200 {object} AssignmentEnvelope
// @Router       /api/v1/assignments/{assignmentId}/publish [post]
func (h *AssessmentHandler) PublishAssignment(c *gin.Context) {
	a, err := h.assessments.PublishAssignment(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("assignmentId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAssignmentResponse(a))
}

// UpsertSubmission godoc
// @Summary      Create or update my assignment submission
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        assignmentId path string true "Assignment ID"
// @Param        body body upsertSubmissionRequest true "Submission"
// @Success      200 {object} SubmissionEnvelope
// @Router       /api/v1/assignments/{assignmentId}/submission [put]
func (h *AssessmentHandler) UpsertSubmission(c *gin.Context) {
	var req upsertSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	sub, err := h.assessments.UpsertSubmission(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("assignmentId"),
		req.Body,
		req.AttachmentURL,
		req.Submit,
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toSubmissionResponse(sub))
}

// GetMySubmission godoc
// @Summary      Get my assignment submission
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        assignmentId path string true "Assignment ID"
// @Success      200 {object} SubmissionEnvelope
// @Router       /api/v1/assignments/{assignmentId}/submission [get]
func (h *AssessmentHandler) GetMySubmission(c *gin.Context) {
	sub, err := h.assessments.GetMySubmission(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("assignmentId"),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toSubmissionResponse(sub))
}

// ListSubmissions godoc
// @Summary      List assignment submissions (teacher)
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        assignmentId path string true "Assignment ID"
// @Success      200 {object} SubmissionListEnvelope
// @Router       /api/v1/assignments/{assignmentId}/submissions [get]
func (h *AssessmentHandler) ListSubmissions(c *gin.Context) {
	items, err := h.assessments.ListSubmissions(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("assignmentId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]SubmissionResponse, 0, len(items))
	for i := range items {
		out = append(out, toSubmissionResponse(&items[i]))
	}
	response.OK(c, out)
}

// GradeSubmission godoc
// @Summary      Grade assignment submission
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        submissionId path string true "Submission ID"
// @Param        body body gradeSubmissionRequest true "Grade"
// @Success      200 {object} SubmissionEnvelope
// @Router       /api/v1/submissions/{submissionId}/grade [post]
func (h *AssessmentHandler) GradeSubmission(c *gin.Context) {
	var req gradeSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	sub, err := h.assessments.GradeSubmission(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("submissionId"),
		req.Score,
		req.Feedback,
		req.RubricScores,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toSubmissionResponse(sub))
}

// Gradebook godoc
// @Summary      Course gradebook
// @Tags         assessments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} GradebookEnvelope
// @Router       /api/v1/courses/{id}/gradebook [get]
func (h *AssessmentHandler) Gradebook(c *gin.Context) {
	items, err := h.assessments.Gradebook(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]GradebookEntryResponse, 0, len(items))
	for i := range items {
		out = append(out, toGradebookEntryResponse(&items[i]))
	}
	response.OK(c, out)
}

func toDomainQuizOptions(opts []quizOptionRequest) []domain.QuizOption {
	out := make([]domain.QuizOption, 0, len(opts))
	for _, o := range opts {
		out = append(out, domain.QuizOption{
			ID:        o.ID,
			Text:      o.Text,
			IsCorrect: o.IsCorrect,
		})
	}
	return out
}

func toDomainRubric(rubric []rubricCriterionRequest) []domain.RubricCriterion {
	out := make([]domain.RubricCriterion, 0, len(rubric))
	for _, r := range rubric {
		out = append(out, domain.RubricCriterion{
			ID:        r.ID,
			Criterion: r.Criterion,
			MaxPoints: r.MaxPoints,
		})
	}
	return out
}

func toQuizResponse(q *domain.Quiz) QuizResponse {
	questions := make([]QuestionResponse, 0, len(q.Questions))
	for i := range q.Questions {
		questions = append(questions, toQuestionResponse(&q.Questions[i]))
	}
	out := QuizResponse{
		ID:               q.ID,
		CourseID:         q.CourseID,
		Title:            q.Title,
		Description:      q.Description,
		Status:           string(q.Status),
		TimeLimitSeconds: q.TimeLimitSeconds,
		MaxAttempts:      q.MaxAttempts,
		PassPercent:      q.PassPercent,
		ShuffleQuestions: q.ShuffleQuestions,
		CreatedAt:        q.CreatedAt,
		UpdatedAt:        q.UpdatedAt,
	}
	if len(questions) > 0 {
		out.Questions = questions
	}
	return out
}

func toQuestionResponse(q *domain.QuizQuestion) QuestionResponse {
	options := make([]QuizOptionResponse, 0, len(q.Options))
	for _, o := range q.Options {
		options = append(options, QuizOptionResponse{
			ID:        o.ID,
			Text:      o.Text,
			IsCorrect: o.IsCorrect,
		})
	}
	out := QuestionResponse{
		ID:            q.ID,
		QuizID:        q.QuizID,
		QuestionType:  string(q.QuestionType),
		Prompt:        q.Prompt,
		Points:        q.Points,
		Position:      q.Position,
		CorrectAnswer: q.CorrectAnswer,
		CreatedAt:     q.CreatedAt,
		UpdatedAt:     q.UpdatedAt,
	}
	if len(options) > 0 {
		out.Options = options
	}
	return out
}

func toAttemptResponse(a *domain.QuizAttempt) AttemptResponse {
	answers := make([]AttemptAnswerResponse, 0, len(a.Answers))
	for i := range a.Answers {
		ans := a.Answers[i]
		ids := ans.SelectedOptionIDs
		if ids == nil {
			ids = []string{}
		}
		answers = append(answers, AttemptAnswerResponse{
			ID:                ans.ID,
			AttemptID:         ans.AttemptID,
			QuestionID:        ans.QuestionID,
			SelectedOptionIDs: ids,
			TextAnswer:        ans.TextAnswer,
			IsCorrect:         ans.IsCorrect,
			PointsAwarded:     ans.PointsAwarded,
			CreatedAt:         ans.CreatedAt,
			UpdatedAt:         ans.UpdatedAt,
		})
	}
	out := AttemptResponse{
		ID:            a.ID,
		QuizID:        a.QuizID,
		UserID:        a.UserID,
		AttemptNumber: a.AttemptNumber,
		Status:        string(a.Status),
		ScorePoints:   a.ScorePoints,
		MaxPoints:     a.MaxPoints,
		Percent:       a.Percent,
		Passed:        a.Passed,
		StartedAt:     a.StartedAt,
		SubmittedAt:   a.SubmittedAt,
		GradedAt:      a.GradedAt,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
		UserEmail:     a.UserEmail,
		UserFullName:  a.UserFullName,
	}
	if len(answers) > 0 {
		out.Answers = answers
	}
	return out
}

func toAssignmentResponse(a *domain.Assignment) AssignmentResponse {
	rubric := make([]RubricCriterionResponse, 0, len(a.Rubric))
	for _, r := range a.Rubric {
		rubric = append(rubric, RubricCriterionResponse{
			ID:        r.ID,
			Criterion: r.Criterion,
			MaxPoints: r.MaxPoints,
		})
	}
	out := AssignmentResponse{
		ID:          a.ID,
		CourseID:    a.CourseID,
		Title:       a.Title,
		Description: a.Description,
		Status:      string(a.Status),
		MaxScore:    a.MaxScore,
		DueAt:       a.DueAt,
		AllowLate:   a.AllowLate,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
	if len(rubric) > 0 {
		out.Rubric = rubric
	}
	return out
}

func toSubmissionResponse(s *domain.AssignmentSubmission) SubmissionResponse {
	return SubmissionResponse{
		ID:            s.ID,
		AssignmentID:  s.AssignmentID,
		UserID:        s.UserID,
		Status:        string(s.Status),
		Body:          s.Body,
		AttachmentURL: s.AttachmentURL,
		Score:         s.Score,
		Feedback:      s.Feedback,
		RubricScores:  s.RubricScores,
		SubmittedAt:   s.SubmittedAt,
		GradedAt:      s.GradedAt,
		GradedBy:      s.GradedBy,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		UserEmail:     s.UserEmail,
		UserFullName:  s.UserFullName,
	}
}

func toGradebookEntryResponse(e *domain.GradebookEntry) GradebookEntryResponse {
	quizzes := make([]GradebookQuizScoreResponse, 0, len(e.Quizzes))
	for _, q := range e.Quizzes {
		quizzes = append(quizzes, GradebookQuizScoreResponse{
			QuizID:    q.QuizID,
			QuizTitle: q.QuizTitle,
			Percent:   q.Percent,
			Passed:    q.Passed,
			Attempts:  q.Attempts,
		})
	}
	assignments := make([]GradebookAssignmentScoreResponse, 0, len(e.Assignments))
	for _, a := range e.Assignments {
		assignments = append(assignments, GradebookAssignmentScoreResponse{
			AssignmentID:    a.AssignmentID,
			AssignmentTitle: a.AssignmentTitle,
			Score:           a.Score,
			MaxScore:        a.MaxScore,
			Status:          a.Status,
		})
	}
	return GradebookEntryResponse{
		UserID:       e.UserID,
		UserEmail:    e.UserEmail,
		UserFullName: e.UserFullName,
		Quizzes:      quizzes,
		Assignments:  assignments,
	}
}
