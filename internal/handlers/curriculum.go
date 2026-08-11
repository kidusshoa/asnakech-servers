package handlers

import (
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type CurriculumHandler struct {
	curriculum *service.CurriculumService
}

func NewCurriculumHandler(curriculum *service.CurriculumService) *CurriculumHandler {
	return &CurriculumHandler{curriculum: curriculum}
}

type createModuleRequest struct {
	Title string `json:"title" binding:"required"`
}

type updateModuleRequest struct {
	Title string `json:"title" binding:"required"`
}

type createLessonRequest struct {
	Title                string  `json:"title" binding:"required"`
	Slug                 string  `json:"slug"`
	Summary              string  `json:"summary"`
	EstimatedMinutes     int     `json:"estimated_minutes"`
	PrerequisiteLessonID *string `json:"prerequisite_lesson_id"`
}

type updateLessonRequest struct {
	Title                string  `json:"title" binding:"required"`
	Summary              string  `json:"summary"`
	EstimatedMinutes     int     `json:"estimated_minutes"`
	PrerequisiteLessonID *string `json:"prerequisite_lesson_id"`
}

type createBlockRequest struct {
	BlockType string  `json:"block_type" binding:"required" example:"text"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	MediaURL  string  `json:"media_url"`
	QuizRefID *string `json:"quiz_ref_id"`
}

type updateBlockRequest struct {
	BlockType string  `json:"block_type" binding:"required" example:"text"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	MediaURL  string  `json:"media_url"`
	QuizRefID *string `json:"quiz_ref_id"`
}

type reorderRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type ModuleResponse struct {
	ID        string           `json:"id"`
	CourseID  string           `json:"course_id"`
	Title     string           `json:"title"`
	Position  int              `json:"position"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Lessons   []LessonResponse `json:"lessons,omitempty"`
}

type LessonResponse struct {
	ID                   string                 `json:"id"`
	ModuleID             string                 `json:"module_id"`
	Title                string                 `json:"title"`
	Slug                 string                 `json:"slug"`
	Summary              string                 `json:"summary"`
	Status               string                 `json:"status"`
	Position             int                    `json:"position"`
	PrerequisiteLessonID *string                `json:"prerequisite_lesson_id,omitempty"`
	EstimatedMinutes     int                    `json:"estimated_minutes"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	Blocks               []ContentBlockResponse `json:"blocks,omitempty"`
}

type ContentBlockResponse struct {
	ID        string    `json:"id"`
	LessonID  string    `json:"lesson_id"`
	BlockType string    `json:"block_type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	MediaURL  string    `json:"media_url"`
	QuizRefID *string   `json:"quiz_ref_id,omitempty"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CurriculumTreeResponse struct {
	CourseID string           `json:"course_id"`
	Modules  []ModuleResponse `json:"modules"`
}

type CurriculumTreeEnvelope struct {
	Success bool                   `json:"success" example:"true"`
	Data    CurriculumTreeResponse `json:"data"`
}

type ModuleEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    ModuleResponse `json:"data"`
}

type LessonEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    LessonResponse `json:"data"`
}

type ContentBlockEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    ContentBlockResponse `json:"data"`
}

// GetCurriculum godoc
// @Summary      Get course curriculum tree
// @Tags         curriculum
// @Produce      json
// @Param        id path string true "Course ID"
// @Success      200 {object} CurriculumTreeEnvelope
// @Router       /api/v1/courses/{id}/curriculum [get]
func (h *CurriculumHandler) GetCurriculum(c *gin.Context) {
	tree, err := h.curriculum.GetTree(
		c.Request.Context(),
		c.Param("id"),
		c.GetString(middleware.ContextUserID),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCurriculumTreeResponse(tree))
}

// CreateModule godoc
// @Summary      Create module
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body createModuleRequest true "Module"
// @Success      201 {object} ModuleEnvelope
// @Router       /api/v1/courses/{id}/modules [post]
func (h *CurriculumHandler) CreateModule(c *gin.Context) {
	var req createModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	mod, err := h.curriculum.CreateModule(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.Title,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toModuleResponse(mod))
}

// UpdateModule godoc
// @Summary      Update module
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        moduleId path string true "Module ID"
// @Param        body body updateModuleRequest true "Module"
// @Success      200 {object} ModuleEnvelope
// @Router       /api/v1/modules/{moduleId} [patch]
func (h *CurriculumHandler) UpdateModule(c *gin.Context) {
	var req updateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	mod, err := h.curriculum.UpdateModule(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("moduleId"),
		req.Title,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toModuleResponse(mod))
}

// DeleteModule godoc
// @Summary      Delete module
// @Tags         curriculum
// @Produce      json
// @Security     BearerAuth
// @Param        moduleId path string true "Module ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/modules/{moduleId} [delete]
func (h *CurriculumHandler) DeleteModule(c *gin.Context) {
	if err := h.curriculum.DeleteModule(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("moduleId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "module deleted"})
}

// ReorderModules godoc
// @Summary      Reorder modules
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body reorderRequest true "Ordered module IDs"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/courses/{id}/modules/reorder [put]
func (h *CurriculumHandler) ReorderModules(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	if err := h.curriculum.ReorderModules(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		req.IDs,
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "modules reordered"})
}

// CreateLesson godoc
// @Summary      Create lesson
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        moduleId path string true "Module ID"
// @Param        body body createLessonRequest true "Lesson"
// @Success      201 {object} LessonEnvelope
// @Router       /api/v1/modules/{moduleId}/lessons [post]
func (h *CurriculumHandler) CreateLesson(c *gin.Context) {
	var req createLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	lesson, err := h.curriculum.CreateLesson(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("moduleId"),
		req.Title,
		req.Slug,
		req.Summary,
		req.EstimatedMinutes,
		req.PrerequisiteLessonID,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toLessonResponse(lesson))
}

// UpdateLesson godoc
// @Summary      Update lesson
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Param        body body updateLessonRequest true "Lesson"
// @Success      200 {object} LessonEnvelope
// @Router       /api/v1/lessons/{lessonId} [patch]
func (h *CurriculumHandler) UpdateLesson(c *gin.Context) {
	var req updateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	lesson, err := h.curriculum.UpdateLesson(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
		req.Title,
		req.Summary,
		req.EstimatedMinutes,
		req.PrerequisiteLessonID,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLessonResponse(lesson))
}

// PublishLesson godoc
// @Summary      Publish lesson
// @Tags         curriculum
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Success      200 {object} LessonEnvelope
// @Router       /api/v1/lessons/{lessonId}/publish [post]
func (h *CurriculumHandler) PublishLesson(c *gin.Context) {
	lesson, err := h.curriculum.PublishLesson(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLessonResponse(lesson))
}

// UnpublishLesson godoc
// @Summary      Unpublish lesson (back to draft)
// @Tags         curriculum
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Success      200 {object} LessonEnvelope
// @Router       /api/v1/lessons/{lessonId}/unpublish [post]
func (h *CurriculumHandler) UnpublishLesson(c *gin.Context) {
	lesson, err := h.curriculum.UnpublishLesson(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toLessonResponse(lesson))
}

// DeleteLesson godoc
// @Summary      Delete lesson
// @Tags         curriculum
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/lessons/{lessonId} [delete]
func (h *CurriculumHandler) DeleteLesson(c *gin.Context) {
	if err := h.curriculum.DeleteLesson(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "lesson deleted"})
}

// ReorderLessons godoc
// @Summary      Reorder lessons in a module
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        moduleId path string true "Module ID"
// @Param        body body reorderRequest true "Ordered lesson IDs"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/modules/{moduleId}/lessons/reorder [put]
func (h *CurriculumHandler) ReorderLessons(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	if err := h.curriculum.ReorderLessons(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("moduleId"),
		req.IDs,
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "lessons reordered"})
}

// CreateBlock godoc
// @Summary      Create content block
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Param        body body createBlockRequest true "Block"
// @Success      201 {object} ContentBlockEnvelope
// @Router       /api/v1/lessons/{lessonId}/blocks [post]
func (h *CurriculumHandler) CreateBlock(c *gin.Context) {
	var req createBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	block, err := h.curriculum.CreateBlock(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
		domain.ContentBlockType(req.BlockType),
		req.Title,
		req.Body,
		req.MediaURL,
		req.QuizRefID,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toBlockResponse(block))
}

// UpdateBlock godoc
// @Summary      Update content block
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        blockId path string true "Block ID"
// @Param        body body updateBlockRequest true "Block"
// @Success      200 {object} ContentBlockEnvelope
// @Router       /api/v1/blocks/{blockId} [patch]
func (h *CurriculumHandler) UpdateBlock(c *gin.Context) {
	var req updateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	block, err := h.curriculum.UpdateBlock(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("blockId"),
		domain.ContentBlockType(req.BlockType),
		req.Title,
		req.Body,
		req.MediaURL,
		req.QuizRefID,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toBlockResponse(block))
}

// DeleteBlock godoc
// @Summary      Delete content block
// @Tags         curriculum
// @Produce      json
// @Security     BearerAuth
// @Param        blockId path string true "Block ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/blocks/{blockId} [delete]
func (h *CurriculumHandler) DeleteBlock(c *gin.Context) {
	if err := h.curriculum.DeleteBlock(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("blockId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "content block deleted"})
}

// ReorderBlocks godoc
// @Summary      Reorder content blocks in a lesson
// @Tags         curriculum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lessonId path string true "Lesson ID"
// @Param        body body reorderRequest true "Ordered block IDs"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/lessons/{lessonId}/blocks/reorder [put]
func (h *CurriculumHandler) ReorderBlocks(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	if err := h.curriculum.ReorderBlocks(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("lessonId"),
		req.IDs,
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "content blocks reordered"})
}

func toCurriculumTreeResponse(t *domain.CurriculumTree) CurriculumTreeResponse {
	modules := make([]ModuleResponse, 0, len(t.Modules))
	for i := range t.Modules {
		modules = append(modules, toModuleResponse(&t.Modules[i]))
	}
	return CurriculumTreeResponse{CourseID: t.CourseID, Modules: modules}
}

func toModuleResponse(m *domain.CourseModule) ModuleResponse {
	lessons := make([]LessonResponse, 0, len(m.Lessons))
	for i := range m.Lessons {
		lessons = append(lessons, toLessonResponse(&m.Lessons[i]))
	}
	out := ModuleResponse{
		ID:        m.ID,
		CourseID:  m.CourseID,
		Title:     m.Title,
		Position:  m.Position,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if len(lessons) > 0 {
		out.Lessons = lessons
	}
	return out
}

func toLessonResponse(l *domain.Lesson) LessonResponse {
	blocks := make([]ContentBlockResponse, 0, len(l.Blocks))
	for i := range l.Blocks {
		blocks = append(blocks, toBlockResponse(&l.Blocks[i]))
	}
	out := LessonResponse{
		ID:                   l.ID,
		ModuleID:             l.ModuleID,
		Title:                l.Title,
		Slug:                 l.Slug,
		Summary:              l.Summary,
		Status:               string(l.Status),
		Position:             l.Position,
		PrerequisiteLessonID: l.PrerequisiteLessonID,
		EstimatedMinutes:     l.EstimatedMinutes,
		CreatedAt:            l.CreatedAt,
		UpdatedAt:            l.UpdatedAt,
	}
	if len(blocks) > 0 {
		out.Blocks = blocks
	}
	return out
}

func toBlockResponse(b *domain.ContentBlock) ContentBlockResponse {
	return ContentBlockResponse{
		ID:        b.ID,
		LessonID:  b.LessonID,
		BlockType: string(b.BlockType),
		Title:     b.Title,
		Body:      b.Body,
		MediaURL:  b.MediaURL,
		QuizRefID: b.QuizRefID,
		Position:  b.Position,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}
