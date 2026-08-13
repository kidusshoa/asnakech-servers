package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
	"github.com/asnakech/asnakech-servers/internal/storage"
)

type MediaService struct {
	media      repository.MediaRepository
	courses    repository.CourseRepository
	store      storage.ObjectStore
	scanner    storage.VirusScanner
	presignTTL time.Duration
}

func NewMediaService(
	media repository.MediaRepository,
	courses repository.CourseRepository,
	store storage.ObjectStore,
	scanner storage.VirusScanner,
	presignTTL time.Duration,
) *MediaService {
	if store == nil {
		store = storage.NoopStore{}
	}
	if scanner == nil {
		scanner = storage.SkipScanner{}
	}
	if presignTTL <= 0 {
		presignTTL = 15 * time.Minute
	}
	return &MediaService{
		media:      media,
		courses:    courses,
		store:      store,
		scanner:    scanner,
		presignTTL: presignTTL,
	}
}

func (s *MediaService) CreateUpload(ctx context.Context, ownerID string, in domain.MediaCreateInput, platformAdmin bool) (*domain.MediaUploadIntent, error) {
	if !s.store.Configured() {
		return nil, apperr.Validation("object storage is not configured")
	}
	purpose := in.Purpose
	if purpose == "" {
		purpose = domain.MediaPurposeGeneral
	}
	limit, ok := storage.LimitFor(purpose)
	if !ok {
		return nil, apperr.Validation("invalid purpose")
	}
	contentType := strings.ToLower(strings.TrimSpace(in.ContentType))
	if contentType == "" {
		return nil, apperr.Validation("content_type is required")
	}
	if !storage.AllowedContentType(purpose, contentType) {
		return nil, apperr.Validation("content_type not allowed for this purpose")
	}
	if in.SizeBytes < 0 {
		return nil, apperr.Validation("size_bytes must be >= 0")
	}
	if in.SizeBytes > limit.MaxBytes {
		return nil, apperr.Validation(fmt.Sprintf("declared size exceeds limit of %d bytes", limit.MaxBytes))
	}
	if in.CourseID != nil && *in.CourseID != "" {
		course, err := s.courses.GetByID(ctx, *in.CourseID)
		if err != nil {
			return nil, err
		}
		if !platformAdmin && course.TeacherID != ownerID {
			// Enrollees may attach assignment files; teachers may attach course media.
			if purpose != domain.MediaPurposeAssignmentAttachment && purpose != domain.MediaPurposeGeneral {
				return nil, apperr.Forbidden("only the course teacher can upload this media type for the course")
			}
		}
	} else {
		in.CourseID = nil
	}

	id := randomHex(16)
	ext := path.Ext(in.OriginalFilename)
	key := fmt.Sprintf("%s/%s/%s%s", purpose, ownerID, id, ext)

	asset := &domain.MediaAsset{
		OwnerID:          ownerID,
		CourseID:         in.CourseID,
		Purpose:          purpose,
		Status:           domain.MediaStatusPending,
		ContentType:      contentType,
		OriginalFilename: strings.TrimSpace(in.OriginalFilename),
		SizeBytes:        in.SizeBytes,
		StorageKey:       key,
		PublicURL:        s.store.PublicURL(key),
		ScanStatus:       domain.MediaScanPending,
	}
	if err := s.media.Create(ctx, asset); err != nil {
		return nil, err
	}

	uploadURL, err := s.store.PresignPut(ctx, key, contentType, s.presignTTL)
	if err != nil {
		return nil, apperr.Internal("failed to create upload URL")
	}

	return &domain.MediaUploadIntent{
		Asset:     *asset,
		Method:    "PUT",
		UploadURL: uploadURL,
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: time.Now().UTC().Add(s.presignTTL),
	}, nil
}

func (s *MediaService) Complete(ctx context.Context, actorID, assetID string, in domain.MediaCompleteInput, platformAdmin bool) (*domain.MediaAsset, error) {
	asset, err := s.requireOwner(ctx, actorID, assetID, platformAdmin)
	if err != nil {
		return nil, err
	}
	if asset.Status != domain.MediaStatusPending && asset.Status != domain.MediaStatusUploaded {
		return nil, apperr.Conflict("media asset is not awaiting completion")
	}

	size, ct, err := s.store.Head(ctx, asset.StorageKey)
	if err != nil {
		return nil, apperr.Validation("uploaded object not found in storage; complete the PUT first")
	}
	limit, _ := storage.LimitFor(asset.Purpose)
	if size > limit.MaxBytes {
		_ = s.store.Delete(ctx, asset.StorageKey)
		asset.Status = domain.MediaStatusRejected
		asset.ScanNote = "exceeded size limit after upload"
		_, _ = s.media.Update(ctx, asset)
		return nil, apperr.Validation("uploaded object exceeds size limit")
	}
	if in.SizeBytes != nil {
		size = *in.SizeBytes
	}
	if ct != "" {
		asset.ContentType = ct
	}
	asset.SizeBytes = size
	asset.DurationSeconds = in.DurationSeconds
	asset.Width = in.Width
	asset.Height = in.Height
	asset.PublicURL = s.store.PublicURL(asset.StorageKey)
	asset.Status = domain.MediaStatusUploaded

	scanStatus, note, err := s.scanner.ScanAsset(ctx, asset)
	if err != nil {
		return nil, err
	}
	asset.ScanStatus = scanStatus
	asset.ScanNote = note
	switch scanStatus {
	case domain.MediaScanInfected:
		asset.Status = domain.MediaStatusRejected
	case domain.MediaScanClean, domain.MediaScanSkipped:
		asset.Status = domain.MediaStatusReady
	default:
		asset.Status = domain.MediaStatusUploaded
	}

	return s.media.Update(ctx, asset)
}

func (s *MediaService) Get(ctx context.Context, actorID, assetID string, platformAdmin bool) (*domain.MediaAsset, error) {
	asset, err := s.media.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if platformAdmin || asset.OwnerID == actorID {
		return asset, nil
	}
	if asset.CourseID != nil {
		course, err := s.courses.GetByID(ctx, *asset.CourseID)
		if err == nil && course.TeacherID == actorID {
			return asset, nil
		}
	}
	return nil, apperr.Forbidden("cannot view this media asset")
}

func (s *MediaService) ListMine(ctx context.Context, actorID string, page, perPage int) ([]domain.MediaAsset, int64, error) {
	return s.media.ListByOwner(ctx, actorID, page, perPage)
}

func (s *MediaService) Delete(ctx context.Context, actorID, assetID string, platformAdmin bool) error {
	asset, err := s.requireOwner(ctx, actorID, assetID, platformAdmin)
	if err != nil {
		return err
	}
	_ = s.store.Delete(ctx, asset.StorageKey)
	return s.media.SoftDelete(ctx, assetID)
}

// ApplyScanResult is the virus-scan worker/webhook hook.
func (s *MediaService) ApplyScanResult(ctx context.Context, actorID, assetID string, status domain.MediaScanStatus, note string, platformAdmin bool) (*domain.MediaAsset, error) {
	if !platformAdmin {
		return nil, apperr.Forbidden("only admins can apply scan results")
	}
	asset, err := s.media.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	switch status {
	case domain.MediaScanClean, domain.MediaScanInfected, domain.MediaScanSkipped, domain.MediaScanPending:
	default:
		return nil, apperr.Validation("invalid scan_status")
	}
	asset.ScanStatus = status
	asset.ScanNote = strings.TrimSpace(note)
	if status == domain.MediaScanInfected {
		asset.Status = domain.MediaStatusRejected
		_ = s.store.Delete(ctx, asset.StorageKey)
	} else if status == domain.MediaScanClean || status == domain.MediaScanSkipped {
		if asset.Status == domain.MediaStatusUploaded || asset.Status == domain.MediaStatusPending {
			asset.Status = domain.MediaStatusReady
		}
	}
	_ = actorID
	return s.media.Update(ctx, asset)
}

func (s *MediaService) requireOwner(ctx context.Context, actorID, assetID string, platformAdmin bool) (*domain.MediaAsset, error) {
	asset, err := s.media.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if platformAdmin || asset.OwnerID == actorID {
		return asset, nil
	}
	return nil, apperr.Forbidden("cannot modify this media asset")
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// AvatarUploadIntent creates a pending avatar upload (replaces Stage 7 stub).
func (s *MediaService) AvatarUploadIntent(ctx context.Context, userID string) (*domain.MediaUploadIntent, error) {
	return s.CreateUpload(ctx, userID, domain.MediaCreateInput{
		Purpose:          domain.MediaPurposeAvatar,
		ContentType:      "image/jpeg",
		OriginalFilename: "avatar.jpg",
		SizeBytes:        5 << 20,
	}, false)
}
