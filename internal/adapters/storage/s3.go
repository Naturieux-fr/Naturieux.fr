package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config configures an S3-compatible object store (AWS S3, MinIO, …).
type S3Config struct {
	Endpoint  string // host[:port], e.g. "localhost:9000" or "s3.amazonaws.com"
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string // default "us-east-1"
	UseSSL    bool
	PublicURL string // public base for object URLs, e.g. "https://cdn.naturieux.fr"
}

// S3 stores media in an S3-compatible bucket and serves it via the bucket's
// public URL (the application does not proxy the files).
type S3 struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

// NewS3 connects to the object store and ensures the bucket exists.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 storage: endpoint and bucket are required")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 storage: connecting: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("s3 storage: checking bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: region}); err != nil {
			return nil, fmt.Errorf("s3 storage: creating bucket: %w", err)
		}
	}

	publicURL := cfg.PublicURL
	if publicURL == "" {
		// Fall back to a path-style URL against the endpoint.
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s/%s", scheme, cfg.Endpoint, cfg.Bucket)
	}
	publicURL = strings.TrimRight(publicURL, "/")

	return &S3{client: client, bucket: cfg.Bucket, publicURL: publicURL}, nil
}

// Save streams the data to the bucket under a generated key and returns its
// public URL.
func (s *S3) Save(ctx context.Context, contentType string, r io.Reader) (Saved, error) {
	ext, err := extensionForAny(contentType)
	if err != nil {
		return Saved{}, err
	}
	key := uuid.New().String() + ext

	// size -1 streams the object without buffering it all in memory.
	_, err = s.client.PutObject(ctx, s.bucket, key, r, -1, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return Saved{}, fmt.Errorf("s3 storage: uploading: %w", err)
	}
	return Saved{URL: s.publicURL + "/" + key}, nil
}

// Delete removes an object addressed by its public URL. URLs not served from
// this bucket are ignored.
func (s *S3) Delete(ctx context.Context, url string) error {
	key, ok := strings.CutPrefix(url, s.publicURL+"/")
	if !ok || key == "" || strings.Contains(key, "/") {
		return nil
	}
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("s3 storage: removing object: %w", err)
	}
	return nil
}

// Ensure interface compliance.
var _ Storage = (*S3)(nil)
