package handlers

import (
	"net/http"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type CertificateHandler struct {
	certs *service.CertificateService
}

func NewCertificateHandler(certs *service.CertificateService) *CertificateHandler {
	return &CertificateHandler{certs: certs}
}

type issueCertificateRequest struct {
	UserID *string `json:"user_id"`
}

type CertificateResponse struct {
	ID               string  `json:"id"`
	CourseID         string  `json:"course_id"`
	UserID           string  `json:"user_id"`
	VerificationCode string  `json:"verification_code"`
	LearnerName      string  `json:"learner_name"`
	CourseTitle      string  `json:"course_title"`
	PublicURL        string  `json:"public_url,omitempty"`
	CourseSlug       string  `json:"course_slug,omitempty"`
	UserEmail        string  `json:"user_email,omitempty"`
	IssuedAt         string  `json:"issued_at"`
	RevokedAt        *string `json:"revoked_at,omitempty"`
}

type CertificateVerifyResponse struct {
	Valid            bool    `json:"valid"`
	VerificationCode string  `json:"verification_code"`
	LearnerName      string  `json:"learner_name,omitempty"`
	CourseTitle      string  `json:"course_title,omitempty"`
	IssuedAt         *string `json:"issued_at,omitempty"`
	RevokedAt        *string `json:"revoked_at,omitempty"`
}

type TranscriptResponse struct {
	UserID       string                   `json:"user_id"`
	UserEmail    string                   `json:"user_email"`
	UserFullName string                   `json:"user_full_name"`
	GeneratedAt  string                   `json:"generated_at"`
	Courses      []TranscriptCourseResponse `json:"courses"`
}

type TranscriptCourseResponse struct {
	CourseID        string                    `json:"course_id"`
	CourseTitle     string                    `json:"course_title"`
	CourseSlug      string                    `json:"course_slug"`
	ProgressPercent int                       `json:"progress_percent"`
	CompletedAt     *string                   `json:"completed_at,omitempty"`
	Quizzes         []transcriptQuizScore     `json:"quizzes"`
	Assignments     []transcriptAssignmentScore `json:"assignments"`
	Certificate     *transcriptCertificateRef `json:"certificate,omitempty"`
}

type transcriptQuizScore struct {
	QuizID    string `json:"quiz_id"`
	QuizTitle string `json:"quiz_title"`
	Percent   *int   `json:"percent,omitempty"`
	Passed    *bool  `json:"passed,omitempty"`
	Attempts  int    `json:"attempts"`
}

type transcriptAssignmentScore struct {
	AssignmentID    string `json:"assignment_id"`
	AssignmentTitle string `json:"assignment_title"`
	Score           *int   `json:"score,omitempty"`
	MaxScore        int    `json:"max_score"`
	Status          string `json:"status,omitempty"`
}

type transcriptCertificateRef struct {
	ID               string `json:"id"`
	VerificationCode string `json:"verification_code"`
	IssuedAt         string `json:"issued_at"`
	Revoked          bool   `json:"revoked"`
}

// IssueCertificate godoc
// @Summary      Issue completion certificate
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body issueCertificateRequest false "Optional user_id for teacher issue"
// @Success      201 {object} CertificateEnvelope
// @Router       /api/v1/courses/{id}/certificate [post]
func (h *CertificateHandler) IssueCertificate(c *gin.Context) {
	var req issueCertificateRequest
	_ = c.ShouldBindJSON(&req)
	cert, err := h.certs.Issue(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), req.UserID, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toCertificateResponse(cert))
}

// ListMyCertificates godoc
// @Summary      List my certificates
// @Tags         certificates
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} CertificateListEnvelope
// @Router       /api/v1/me/certificates [get]
func (h *CertificateHandler) ListMyCertificates(c *gin.Context) {
	items, err := h.certs.ListMine(c.Request.Context(), c.GetString(middleware.ContextUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]CertificateResponse, 0, len(items))
	for i := range items {
		out = append(out, toCertificateResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Total: int64(len(out))})
}

// ListCourseCertificates godoc
// @Summary      List certificates issued for a course
// @Tags         certificates
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} CertificateListEnvelope
// @Router       /api/v1/courses/{id}/certificates [get]
func (h *CertificateHandler) ListCourseCertificates(c *gin.Context) {
	items, err := h.certs.ListCourseCertificates(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]CertificateResponse, 0, len(items))
	for i := range items {
		out = append(out, toCertificateResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Total: int64(len(out))})
}

// GetCertificate godoc
// @Summary      Get certificate
// @Tags         certificates
// @Produce      json
// @Security     BearerAuth
// @Param        certificateId path string true "Certificate ID"
// @Success      200 {object} CertificateEnvelope
// @Router       /api/v1/certificates/{certificateId} [get]
func (h *CertificateHandler) GetCertificate(c *gin.Context) {
	cert, err := h.certs.Get(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("certificateId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCertificateResponse(cert))
}

// DownloadCertificate godoc
// @Summary      Download certificate PDF
// @Tags         certificates
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        certificateId path string true "Certificate ID"
// @Success      200 {file} binary
// @Router       /api/v1/certificates/{certificateId}/download [get]
func (h *CertificateHandler) DownloadCertificate(c *gin.Context) {
	pdf, filename, err := h.certs.DownloadPDF(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("certificateId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// RevokeCertificate godoc
// @Summary      Revoke certificate
// @Tags         certificates
// @Produce      json
// @Security     BearerAuth
// @Param        certificateId path string true "Certificate ID"
// @Success      200 {object} CertificateEnvelope
// @Router       /api/v1/certificates/{certificateId}/revoke [post]
func (h *CertificateHandler) RevokeCertificate(c *gin.Context) {
	cert, err := h.certs.Revoke(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("certificateId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCertificateResponse(cert))
}

// VerifyCertificate godoc
// @Summary      Verify certificate by code (public)
// @Tags         certificates
// @Produce      json
// @Param        code path string true "Verification code"
// @Success      200 {object} CertificateVerifyEnvelope
// @Router       /api/v1/certificates/verify/{code} [get]
func (h *CertificateHandler) VerifyCertificate(c *gin.Context) {
	result, err := h.certs.Verify(c.Request.Context(), c.Param("code"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCertificateVerifyResponse(result))
}

// MyTranscript godoc
// @Summary      Export my transcript
// @Tags         certificates
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} TranscriptEnvelope
// @Router       /api/v1/me/transcript [get]
func (h *CertificateHandler) MyTranscript(c *gin.Context) {
	tr, err := h.certs.MyTranscript(c.Request.Context(), c.GetString(middleware.ContextUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toTranscriptResponse(tr))
}

// UserCourseTranscript godoc
// @Summary      Export learner transcript for a course
// @Tags         certificates
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        userId path string true "User ID"
// @Success      200 {object} TranscriptEnvelope
// @Router       /api/v1/courses/{id}/transcript/{userId} [get]
func (h *CertificateHandler) UserCourseTranscript(c *gin.Context) {
	tr, err := h.certs.UserTranscript(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), c.Param("userId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toTranscriptResponse(tr))
}

func toCertificateResponse(c *domain.Certificate) CertificateResponse {
	resp := CertificateResponse{
		ID: c.ID, CourseID: c.CourseID, UserID: c.UserID,
		VerificationCode: c.VerificationCode, LearnerName: c.LearnerName,
		CourseTitle: c.CourseTitle, PublicURL: c.PublicURL,
		CourseSlug: c.CourseSlug, UserEmail: c.UserEmail,
		IssuedAt: c.IssuedAt.Format(timeRFC3339),
	}
	if c.RevokedAt != nil {
		s := c.RevokedAt.Format(timeRFC3339)
		resp.RevokedAt = &s
	}
	return resp
}

func toCertificateVerifyResponse(v *domain.CertificateVerify) CertificateVerifyResponse {
	resp := CertificateVerifyResponse{
		Valid: v.Valid, VerificationCode: v.VerificationCode,
		LearnerName: v.LearnerName, CourseTitle: v.CourseTitle,
	}
	if v.Valid {
		s := v.IssuedAt.Format(timeRFC3339)
		resp.IssuedAt = &s
	}
	if v.RevokedAt != nil {
		s := v.RevokedAt.Format(timeRFC3339)
		resp.RevokedAt = &s
	}
	return resp
}

func toTranscriptResponse(tr *domain.Transcript) TranscriptResponse {
	out := TranscriptResponse{
		UserID: tr.UserID, UserEmail: tr.UserEmail, UserFullName: tr.UserFullName,
		GeneratedAt: tr.GeneratedAt.Format(timeRFC3339),
		Courses:     make([]TranscriptCourseResponse, 0, len(tr.Courses)),
	}
	for _, c := range tr.Courses {
		row := TranscriptCourseResponse{
			CourseID: c.CourseID, CourseTitle: c.CourseTitle, CourseSlug: c.CourseSlug,
			ProgressPercent: c.ProgressPercent,
			Quizzes:         make([]transcriptQuizScore, 0, len(c.Quizzes)),
			Assignments:     make([]transcriptAssignmentScore, 0, len(c.Assignments)),
		}
		for _, q := range c.Quizzes {
			row.Quizzes = append(row.Quizzes, transcriptQuizScore{
				QuizID: q.QuizID, QuizTitle: q.QuizTitle, Percent: q.Percent, Passed: q.Passed, Attempts: q.Attempts,
			})
		}
		for _, a := range c.Assignments {
			row.Assignments = append(row.Assignments, transcriptAssignmentScore{
				AssignmentID: a.AssignmentID, AssignmentTitle: a.AssignmentTitle,
				Score: a.Score, MaxScore: a.MaxScore, Status: a.Status,
			})
		}
		if c.Certificate != nil {
			row.Certificate = &transcriptCertificateRef{
				ID: c.Certificate.ID, VerificationCode: c.Certificate.VerificationCode,
				IssuedAt: c.Certificate.IssuedAt.Format(timeRFC3339), Revoked: c.Certificate.Revoked,
			}
		}
		if c.CompletedAt != nil {
			s := c.CompletedAt.Format(timeRFC3339)
			row.CompletedAt = &s
		}
		out.Courses = append(out.Courses, row)
	}
	return out
}

type CertificateEnvelope struct {
	Success bool                `json:"success" example:"true"`
	Data    CertificateResponse `json:"data"`
}

type CertificateListEnvelope struct {
	Success bool                  `json:"success" example:"true"`
	Data    []CertificateResponse `json:"data"`
	Meta    response.Meta         `json:"meta"`
}

type CertificateVerifyEnvelope struct {
	Success bool                      `json:"success" example:"true"`
	Data    CertificateVerifyResponse `json:"data"`
}

type TranscriptEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    TranscriptResponse `json:"data"`
}
