package storage

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

// VirusScanner is a hook point for asynchronous malware scanning.
type VirusScanner interface {
	// ScanAsset is called after upload complete. Implementations may enqueue work.
	ScanAsset(ctx context.Context, asset *domain.MediaAsset) (domain.MediaScanStatus, string, error)
}

// SkipScanner marks assets as skipped (dev default until a real scanner is wired).
type SkipScanner struct{}

func (SkipScanner) ScanAsset(_ context.Context, _ *domain.MediaAsset) (domain.MediaScanStatus, string, error) {
	return domain.MediaScanSkipped, "virus scan skipped (hook not configured)", nil
}
