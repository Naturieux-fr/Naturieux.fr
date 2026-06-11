package storage_test

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
)

func TestFromEnv_DefaultLocal(t *testing.T) {
	t.Setenv("STORAGE", "")
	t.Setenv("MEDIA_DIR", t.TempDir())
	s, err := storage.FromEnv(context.Background())
	if err != nil {
		t.Fatalf("FromEnv local: %v", err)
	}
	if l, ok := s.(*storage.Local); !ok || l.Dir() == "" {
		t.Errorf("expected a local store with a dir")
	}
}

func TestFromEnv_S3MissingConfig(t *testing.T) {
	t.Setenv("STORAGE", "s3")
	t.Setenv("S3_ENDPOINT", "")
	t.Setenv("S3_BUCKET", "")
	if _, err := storage.FromEnv(context.Background()); err == nil {
		t.Error("FromEnv s3 without config should error")
	}
}
