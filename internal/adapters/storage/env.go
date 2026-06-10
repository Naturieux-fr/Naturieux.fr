package storage

import (
	"context"
	"os"
)

// FromEnv builds a Storage from environment variables. STORAGE selects the
// backend: "s3" for an S3-compatible store (AWS S3 / MinIO), anything else
// for local disk under MEDIA_DIR (default "media").
func FromEnv(ctx context.Context) (Storage, error) {
	if os.Getenv("STORAGE") == "s3" {
		return NewS3(ctx, S3Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			Bucket:    os.Getenv("S3_BUCKET"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
			Region:    os.Getenv("S3_REGION"),
			UseSSL:    os.Getenv("S3_USE_SSL") == "true" || os.Getenv("S3_USE_SSL") == "1",
			PublicURL: os.Getenv("S3_PUBLIC_URL"),
		})
	}

	dir := os.Getenv("MEDIA_DIR")
	if dir == "" {
		dir = "media"
	}
	return NewLocal(dir)
}
