package service

import (
	"context"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type CurriculumService struct {
	courses     repository.CourseRepository
	modules     repository.ModuleRepository
	lessons     repository.LessonRepository
	blocks      repository.ContentBlockRepository
	enrollments *EnrollmentService
}

func NewCurriculumService(
	courses repository.CourseRepository,
	modules repository.ModuleRepository,
	lessons repository.LessonRepository,
	blocks repository.ContentBlockRepository,
	enrollments *EnrollmentService,
) *CurriculumService {
	return &CurriculumService{
		courses:     courses,
		modules:     modules,
		lessons:     lessons,
		blocks:      blocks,
		enrollments: enrollments,
	}
}

func (s *CurriculumService) GetTree(ctx context.Context, courseID, actorID string, platformAdmin bool) (*domain.CurriculumTree, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	canAuthor := platformAdmin || (actorID != "" && course.TeacherID == actorID)
	if course.Status != domain.CourseStatusPublished && !canAuthor {
		return nil, apperr.NotFound("course not found")
	}

	includeBlocks := canAuthor
	if !includeBlocks && s.enrollments != nil {
		ok, err := s.enrollments.HasActiveEnrollment(ctx, courseID, actorID)
		if err != nil {
			return nil, err
		}
		includeBlocks = ok
	}

	modules, err := s.modules.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	tree := &domain.CurriculumTree{CourseID: courseID, Modules: make([]domain.CourseModule, 0, len(modules))}
	for _, mod := range modules {
		lessons, err := s.lessons.ListByModule(ctx, mod.ID)
		if err != nil {
			return nil, err
		}
		visible := make([]domain.Lesson, 0, len(lessons))
		for _, lesson := range lessons {
			if !canAuthor && lesson.Status != domain.LessonStatusPublished {
				continue
			}
			if includeBlocks {
				blocks, err := s.blocks.ListByLesson(ctx, lesson.ID)
				if err != nil {
					return nil, err
				}
				lesson.Blocks = blocks
			}
			visible = append(visible, lesson)
		}
		mod.Lessons = visible
		tree.Modules = append(tree.Modules, mod)
	}
	return tree, nil
}

func (s *CurriculumService) CreateModule(ctx context.Context, actorID, courseID, title string, platformAdmin bool) (*domain.CourseModule, error) {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	pos, err := s.modules.NextPosition(ctx, courseID)
	if err != nil {
		return nil, err
	}
	m := &domain.CourseModule{CourseID: courseID, Title: title, Position: pos}
	if err := s.modules.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *CurriculumService) UpdateModule(ctx context.Context, actorID, moduleID, title string, platformAdmin bool) (*domain.CourseModule, error) {
	mod, err := s.modules.GetByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, mod.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	return s.modules.Update(ctx, moduleID, title)
}

func (s *CurriculumService) DeleteModule(ctx context.Context, actorID, moduleID string, platformAdmin bool) error {
	mod, err := s.modules.GetByID(ctx, moduleID)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, mod.CourseID, platformAdmin); err != nil {
		return err
	}
	return s.modules.Delete(ctx, moduleID)
}

func (s *CurriculumService) ReorderModules(ctx context.Context, actorID, courseID string, ids []string, platformAdmin bool) error {
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return err
	}
	return s.modules.Reorder(ctx, courseID, ids)
}

func (s *CurriculumService) CreateLesson(ctx context.Context, actorID, moduleID string, title, slug, summary string, minutes int, prereq *string, platformAdmin bool) (*domain.Lesson, error) {
	mod, err := s.modules.GetByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, mod.CourseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	if strings.TrimSpace(slug) == "" {
		slug = Slugify(title)
	} else {
		slug = Slugify(slug)
	}
	if minutes < 0 {
		return nil, apperr.Validation("estimated_minutes must be >= 0")
	}
	if prereq != nil && *prereq != "" {
		if _, err := s.lessons.GetByID(ctx, *prereq); err != nil {
			return nil, err
		}
	} else {
		prereq = nil
	}
	pos, err := s.lessons.NextPosition(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	l := &domain.Lesson{
		ModuleID:             moduleID,
		Title:                title,
		Slug:                 slug,
		Summary:              strings.TrimSpace(summary),
		Status:               domain.LessonStatusDraft,
		Position:             pos,
		PrerequisiteLessonID: prereq,
		EstimatedMinutes:     minutes,
	}
	if err := s.lessons.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *CurriculumService) UpdateLesson(ctx context.Context, actorID, lessonID, title, summary string, minutes int, prereq *string, platformAdmin bool) (*domain.Lesson, error) {
	_, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	if minutes < 0 {
		return nil, apperr.Validation("estimated_minutes must be >= 0")
	}
	if prereq != nil {
		if *prereq == "" {
			prereq = nil
		} else if *prereq == lessonID {
			return nil, apperr.Validation("lesson cannot prerequisite itself")
		} else if _, err := s.lessons.GetByID(ctx, *prereq); err != nil {
			return nil, err
		}
	}
	return s.lessons.Update(ctx, lessonID, title, strings.TrimSpace(summary), minutes, prereq)
}

func (s *CurriculumService) PublishLesson(ctx context.Context, actorID, lessonID string, platformAdmin bool) (*domain.Lesson, error) {
	_, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.lessons.SetStatus(ctx, lessonID, domain.LessonStatusPublished)
}

func (s *CurriculumService) UnpublishLesson(ctx context.Context, actorID, lessonID string, platformAdmin bool) (*domain.Lesson, error) {
	_, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	return s.lessons.SetStatus(ctx, lessonID, domain.LessonStatusDraft)
}

func (s *CurriculumService) DeleteLesson(ctx context.Context, actorID, lessonID string, platformAdmin bool) error {
	_, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return err
	}
	return s.lessons.Delete(ctx, lessonID)
}

func (s *CurriculumService) ReorderLessons(ctx context.Context, actorID, moduleID string, ids []string, platformAdmin bool) error {
	mod, err := s.modules.GetByID(ctx, moduleID)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, mod.CourseID, platformAdmin); err != nil {
		return err
	}
	return s.lessons.Reorder(ctx, moduleID, ids)
}

func (s *CurriculumService) CreateBlock(ctx context.Context, actorID, lessonID string, blockType domain.ContentBlockType, title, body, mediaURL string, quizRef *string, platformAdmin bool) (*domain.ContentBlock, error) {
	_, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	if !validBlockType(blockType) {
		return nil, apperr.Validation("invalid block_type")
	}
	if quizRef != nil && *quizRef == "" {
		quizRef = nil
	}
	pos, err := s.blocks.NextPosition(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	b := &domain.ContentBlock{
		LessonID:  lessonID,
		BlockType: blockType,
		Title:     strings.TrimSpace(title),
		Body:      body,
		MediaURL:  strings.TrimSpace(mediaURL),
		QuizRefID: quizRef,
		Position:  pos,
	}
	if err := s.blocks.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *CurriculumService) UpdateBlock(ctx context.Context, actorID, blockID string, blockType domain.ContentBlockType, title, body, mediaURL string, quizRef *string, platformAdmin bool) (*domain.ContentBlock, error) {
	block, err := s.blocks.GetByID(ctx, blockID)
	if err != nil {
		return nil, err
	}
	_, courseID, err := s.lessonCourse(ctx, block.LessonID)
	if err != nil {
		return nil, err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return nil, err
	}
	if !validBlockType(blockType) {
		return nil, apperr.Validation("invalid block_type")
	}
	if quizRef != nil && *quizRef == "" {
		quizRef = nil
	}
	block.BlockType = blockType
	block.Title = strings.TrimSpace(title)
	block.Body = body
	block.MediaURL = strings.TrimSpace(mediaURL)
	block.QuizRefID = quizRef
	return s.blocks.Update(ctx, block)
}

func (s *CurriculumService) DeleteBlock(ctx context.Context, actorID, blockID string, platformAdmin bool) error {
	block, err := s.blocks.GetByID(ctx, blockID)
	if err != nil {
		return err
	}
	_, courseID, err := s.lessonCourse(ctx, block.LessonID)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return err
	}
	return s.blocks.Delete(ctx, blockID)
}

func (s *CurriculumService) ReorderBlocks(ctx context.Context, actorID, lessonID string, ids []string, platformAdmin bool) error {
	_, courseID, err := s.lessonCourse(ctx, lessonID)
	if err != nil {
		return err
	}
	if err := s.requireCourseWrite(ctx, actorID, courseID, platformAdmin); err != nil {
		return err
	}
	return s.blocks.Reorder(ctx, lessonID, ids)
}

func (s *CurriculumService) requireCourseWrite(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if platformAdmin {
		return nil
	}
	if course.TeacherID != actorID {
		return apperr.Forbidden("only the course teacher or an admin can edit curriculum")
	}
	return nil
}

func (s *CurriculumService) lessonCourse(ctx context.Context, lessonID string) (*domain.Lesson, string, error) {
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

func validBlockType(t domain.ContentBlockType) bool {
	switch t {
	case domain.ContentBlockText, domain.ContentBlockVideo, domain.ContentBlockFile, domain.ContentBlockQuizRef:
		return true
	default:
		return false
	}
}
