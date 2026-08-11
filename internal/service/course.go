package service

import (
	"context"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type CourseService struct {
	courses repository.CourseRepository
	cats    repository.CategoryRepository
	tags    repository.TagRepository
	orgs    repository.OrganizationRepository
	members repository.OrganizationMemberRepository
}

func NewCourseService(
	courses repository.CourseRepository,
	cats repository.CategoryRepository,
	tags repository.TagRepository,
	orgs repository.OrganizationRepository,
	members repository.OrganizationMemberRepository,
) *CourseService {
	return &CourseService{
		courses: courses,
		cats:    cats,
		tags:    tags,
		orgs:    orgs,
		members: members,
	}
}

func (s *CourseService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.cats.List(ctx)
}

func (s *CourseService) CreateCategory(ctx context.Context, name, slug, description string) (*domain.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Validation("name is required")
	}
	if strings.TrimSpace(slug) == "" {
		slug = Slugify(name)
	} else {
		slug = Slugify(slug)
	}
	cat := &domain.Category{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(description),
	}
	if err := s.cats.Create(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *CourseService) Create(ctx context.Context, teacherID string, in domain.CourseCreate, platformAdmin bool) (*domain.Course, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = Slugify(title)
	} else {
		slug = Slugify(slug)
	}
	if slug == "" {
		return nil, apperr.Validation("slug is required")
	}

	level := in.Level
	if level == "" {
		level = domain.CourseLevelBeginner
	}
	if !validLevel(level) {
		return nil, apperr.Validation("invalid level")
	}

	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "ETB"
	}
	if len(currency) != 3 {
		return nil, apperr.Validation("currency must be a 3-letter code")
	}
	if in.PriceCents < 0 {
		return nil, apperr.Validation("price_cents must be >= 0")
	}

	if in.CategoryID != nil && *in.CategoryID != "" {
		if _, err := s.cats.GetByID(ctx, *in.CategoryID); err != nil {
			return nil, err
		}
	} else {
		in.CategoryID = nil
	}

	if in.OrganizationID != nil && *in.OrganizationID != "" {
		if _, err := s.orgs.GetByID(ctx, *in.OrganizationID); err != nil {
			return nil, err
		}
		if !platformAdmin {
			member, err := s.members.Get(ctx, *in.OrganizationID, teacherID)
			if err != nil {
				return nil, err
			}
			if !member.OrgRole.CanManage() && member.OrgRole != domain.OrgRoleMember {
				return nil, apperr.Forbidden("not allowed to create courses for this organization")
			}
		}
	} else {
		in.OrganizationID = nil
	}

	lang := strings.TrimSpace(in.Language)
	if lang == "" {
		lang = "en"
	}

	course := &domain.Course{
		OrganizationID: in.OrganizationID,
		TeacherID:      teacherID,
		CategoryID:     in.CategoryID,
		Title:          title,
		Slug:           slug,
		Summary:        strings.TrimSpace(in.Summary),
		Description:    strings.TrimSpace(in.Description),
		Status:         domain.CourseStatusDraft,
		CoverURL:       strings.TrimSpace(in.CoverURL),
		PriceCents:     in.PriceCents,
		Currency:       currency,
		Level:          level,
		Language:       lang,
	}
	if err := s.courses.Create(ctx, course); err != nil {
		return nil, err
	}
	if err := s.applyTags(ctx, course.ID, in.Tags); err != nil {
		return nil, err
	}
	return s.hydrate(ctx, course)
}

func (s *CourseService) Get(ctx context.Context, id, actorID string, platformAdmin bool) (*domain.Course, error) {
	course, err := s.courses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.canView(course, actorID, platformAdmin) {
		return nil, apperr.NotFound("course not found")
	}
	return s.hydrate(ctx, course)
}

func (s *CourseService) List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int64, domain.CourseListFilter, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}
	if filter.Level != "" && !validLevel(filter.Level) {
		return nil, 0, filter, apperr.Validation("invalid level filter")
	}
	if filter.Status != "" && !validStatus(filter.Status) {
		return nil, 0, filter, apperr.Validation("invalid status filter")
	}

	courses, total, err := s.courses.List(ctx, filter)
	if err != nil {
		return nil, 0, filter, err
	}
	for i := range courses {
		hydrated, err := s.hydrate(ctx, &courses[i])
		if err != nil {
			return nil, 0, filter, err
		}
		courses[i] = *hydrated
	}
	return courses, total, filter, nil
}

func (s *CourseService) Update(ctx context.Context, actorID, courseID string, patch domain.CourseUpdate, platformAdmin bool) (*domain.Course, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if err := s.requireWrite(course, actorID, platformAdmin); err != nil {
		return nil, err
	}
	if patch.Title != nil && strings.TrimSpace(*patch.Title) == "" {
		return nil, apperr.Validation("title cannot be empty")
	}
	if patch.Level != nil && !validLevel(*patch.Level) {
		return nil, apperr.Validation("invalid level")
	}
	if patch.PriceCents != nil && *patch.PriceCents < 0 {
		return nil, apperr.Validation("price_cents must be >= 0")
	}
	if patch.CategoryID != nil && *patch.CategoryID != "" {
		if _, err := s.cats.GetByID(ctx, *patch.CategoryID); err != nil {
			return nil, err
		}
	}
	updated, err := s.courses.Update(ctx, courseID, patch)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, updated)
}

func (s *CourseService) SetTags(ctx context.Context, actorID, courseID string, tagNames []string, platformAdmin bool) (*domain.Course, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if err := s.requireWrite(course, actorID, platformAdmin); err != nil {
		return nil, err
	}
	if err := s.applyTags(ctx, courseID, tagNames); err != nil {
		return nil, err
	}
	return s.hydrate(ctx, course)
}

func (s *CourseService) Publish(ctx context.Context, actorID, courseID string, platformAdmin bool) (*domain.Course, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if err := s.requireWrite(course, actorID, platformAdmin); err != nil {
		return nil, err
	}
	if course.Status == domain.CourseStatusPublished {
		return s.hydrate(ctx, course)
	}
	now := time.Now().UTC()
	updated, err := s.courses.SetStatus(ctx, courseID, domain.CourseStatusPublished, &now)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, updated)
}

func (s *CourseService) Archive(ctx context.Context, actorID, courseID string, platformAdmin bool) (*domain.Course, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if err := s.requireWrite(course, actorID, platformAdmin); err != nil {
		return nil, err
	}
	updated, err := s.courses.SetStatus(ctx, courseID, domain.CourseStatusArchived, course.PublishedAt)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, updated)
}

func (s *CourseService) Delete(ctx context.Context, actorID, courseID string, platformAdmin bool) error {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return err
	}
	if err := s.requireWrite(course, actorID, platformAdmin); err != nil {
		return err
	}
	return s.courses.SoftDelete(ctx, courseID, time.Now().UTC())
}

func (s *CourseService) applyTags(ctx context.Context, courseID string, names []string) error {
	if names == nil {
		return nil
	}
	tags, err := s.tags.GetOrCreateByNames(ctx, names)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(tags))
	for _, t := range tags {
		ids = append(ids, t.ID)
	}
	return s.tags.ReplaceCourseTags(ctx, courseID, ids)
}

func (s *CourseService) hydrate(ctx context.Context, course *domain.Course) (*domain.Course, error) {
	tags, err := s.tags.ListByCourse(ctx, course.ID)
	if err != nil {
		return nil, err
	}
	course.TagSlugs = make([]string, 0, len(tags))
	for _, t := range tags {
		course.TagSlugs = append(course.TagSlugs, t.Slug)
	}
	return course, nil
}

func (s *CourseService) canView(course *domain.Course, actorID string, platformAdmin bool) bool {
	if course.Status == domain.CourseStatusPublished {
		return true
	}
	if platformAdmin {
		return true
	}
	return actorID != "" && course.TeacherID == actorID
}

func (s *CourseService) requireWrite(course *domain.Course, actorID string, platformAdmin bool) error {
	if platformAdmin {
		return nil
	}
	if course.TeacherID != actorID {
		return apperr.Forbidden("only the course teacher or an admin can modify this course")
	}
	return nil
}

func validLevel(level domain.CourseLevel) bool {
	switch level {
	case domain.CourseLevelBeginner, domain.CourseLevelIntermediate, domain.CourseLevelAdvanced:
		return true
	default:
		return false
	}
}

func validStatus(status domain.CourseStatus) bool {
	switch status {
	case domain.CourseStatusDraft, domain.CourseStatusPublished, domain.CourseStatusArchived:
		return true
	default:
		return false
	}
}
