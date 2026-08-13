package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ObjectStore abstracts S3/MinIO operations used by media uploads.
type ObjectStore interface {
	Configured() bool
	PresignPut(ctx context.Context, key, contentType string, expiry time.Duration) (string, error)
	Head(ctx context.Context, key string) (size int64, contentType string, err error)
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

// NoopStore is used when S3 is not configured.
type NoopStore struct{}

func (NoopStore) Configured() bool { return false }

func (NoopStore) PresignPut(context.Context, string, string, time.Duration) (string, error) {
	return "", fmt.Errorf("object storage is not configured")
}

func (NoopStore) Head(context.Context, string) (int64, string, error) {
	return 0, "", fmt.Errorf("object storage is not configured")
}

func (NoopStore) Delete(context.Context, string) error {
	return fmt.Errorf("object storage is not configured")
}

func (NoopStore) PublicURL(string) string { return "" }

// JoinPublicURL builds a CDN-friendly URL from optional public base or endpoint+bucket.
func JoinPublicURL(publicBase, endpoint, bucket, key string, pathStyle bool) string {
	key = strings.TrimLeft(key, "/")
	if base := strings.TrimRight(strings.TrimSpace(publicBase), "/"); base != "" {
		return base + "/" + key
	}
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" || bucket == "" {
		return ""
	}
	if pathStyle {
		return endpoint + "/" + bucket + "/" + key
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return endpoint + "/" + key
	}
	u.Host = bucket + "." + u.Host
	u.Path = "/" + key
	return u.String()
}
