package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Store implements ObjectStore against S3-compatible storage (MinIO included).
type S3Store struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
	endpoint   string
	publicBase string
	pathStyle  bool
}

// NewS3Store builds an S3 client from app config. Returns NoopStore when unset.
func NewS3Store(cfg *config.Config) (ObjectStore, error) {
	if cfg.S3Endpoint == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" || cfg.S3Bucket == "" {
		return NoopStore{}, nil
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(strings.TrimRight(cfg.S3Endpoint, "/"))
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	return &S3Store{
		client:     client,
		presigner:  s3.NewPresignClient(client),
		bucket:     cfg.S3Bucket,
		endpoint:   cfg.S3Endpoint,
		publicBase: cfg.S3PublicBaseURL,
		pathStyle:  cfg.S3UsePathStyle,
	}, nil
}

func (s *S3Store) Configured() bool { return true }

func (s *S3Store) PresignPut(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	out, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign put: %w", err)
	}
	return out.URL, nil
}

func (s *S3Store) Head(ctx context.Context, key string) (int64, string, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, "", fmt.Errorf("head object: %w", err)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return size, ct, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *S3Store) PublicURL(key string) string {
	return JoinPublicURL(s.publicBase, s.endpoint, s.bucket, strings.TrimLeft(key, "/"), s.pathStyle)
}
