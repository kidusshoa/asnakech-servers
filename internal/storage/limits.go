package storage

import (
	"strings"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

// PurposeLimit constrains uploads by purpose.
type PurposeLimit struct {
	MaxBytes     int64
	AllowedTypes []string // prefix match, e.g. "image/", or exact "application/pdf"
}

var purposeLimits = map[domain.MediaPurpose]PurposeLimit{
	domain.MediaPurposeAvatar: {
		MaxBytes:     5 << 20,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
	},
	domain.MediaPurposeCourseCover: {
		MaxBytes:     10 << 20,
		AllowedTypes: []string{"image/"},
	},
	domain.MediaPurposeLessonMedia: {
		MaxBytes:     500 << 20,
		AllowedTypes: []string{"video/", "audio/", "image/", "application/pdf"},
	},
	domain.MediaPurposeAssignmentAttachment: {
		MaxBytes:     50 << 20,
		AllowedTypes: []string{"application/pdf", "image/", "text/plain", "application/zip"},
	},
	domain.MediaPurposeGeneral: {
		MaxBytes:     25 << 20,
		AllowedTypes: []string{"image/", "application/pdf", "video/", "audio/"},
	},
}

// LimitFor returns the limit for a purpose, or false if unknown.
func LimitFor(purpose domain.MediaPurpose) (PurposeLimit, bool) {
	l, ok := purposeLimits[purpose]
	return l, ok
}

// AllowedContentType reports whether contentType is permitted for the purpose.
func AllowedContentType(purpose domain.MediaPurpose, contentType string) bool {
	limit, ok := purposeLimits[purpose]
	if !ok {
		return false
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	for _, allowed := range limit.AllowedTypes {
		if strings.HasSuffix(allowed, "/") {
			if strings.HasPrefix(ct, allowed) {
				return true
			}
			continue
		}
		if ct == allowed {
			return true
		}
	}
	return false
}
