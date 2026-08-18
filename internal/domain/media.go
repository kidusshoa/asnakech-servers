package domain

import "time"

// MediaPurpose classifies why an asset was uploaded.
type MediaPurpose string

const (
	MediaPurposeAvatar               MediaPurpose = "avatar"
	MediaPurposeCourseCover          MediaPurpose = "course_cover"
	MediaPurposeLessonMedia          MediaPurpose = "lesson_media"
	MediaPurposeAssignmentAttachment MediaPurpose = "assignment_attachment"
	MediaPurposeGeneral              MediaPurpose = "general"
)

// MediaStatus is the asset lifecycle.
type MediaStatus string

const (
	MediaStatusPending  MediaStatus = "pending"
	MediaStatusUploaded MediaStatus = "uploaded"
	MediaStatusReady    MediaStatus = "ready"
	MediaStatusRejected MediaStatus = "rejected"
	MediaStatusDeleted  MediaStatus = "deleted"
)

// MediaScanStatus is the virus-scan hook result.
type MediaScanStatus string

const (
	MediaScanPending  MediaScanStatus = "pending"
	MediaScanClean    MediaScanStatus = "clean"
	MediaScanInfected MediaScanStatus = "infected"
	MediaScanSkipped  MediaScanStatus = "skipped"
)

// MediaAsset is a stored object referenced by CDN-friendly URL (never binary through API).
type MediaAsset struct {
	ID               string
	OwnerID          string
	CourseID         *string
	Purpose          MediaPurpose
	Status           MediaStatus
	ContentType      string
	OriginalFilename string
	SizeBytes        int64
	StorageKey       string
	PublicURL        string
	DurationSeconds  *int
	Width            *int
	Height           *int
	ScanStatus       MediaScanStatus
	ScanNote         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// MediaUploadIntent is returned to the client for a direct-to-storage PUT.
type MediaUploadIntent struct {
	Asset     MediaAsset
	Method    string
	UploadURL string
	Headers   map[string]string
	ExpiresAt time.Time
}

// MediaCompleteInput confirms an upload and optional video metadata.
type MediaCompleteInput struct {
	SizeBytes       *int64
	DurationSeconds *int
	Width           *int
	Height          *int
}

// MediaCreateInput starts a pending upload.
type MediaCreateInput struct {
	Purpose          MediaPurpose
	ContentType      string
	OriginalFilename string
	CourseID         *string
	SizeBytes        int64 // declared size for limit checks before upload
}
