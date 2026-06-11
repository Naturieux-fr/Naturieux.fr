package storage_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
)

func TestS3_RejectsBadTypeBeforeNetwork(t *testing.T) {
	// A bad content type is rejected before any S3 call, so no network needed.
	s := &storage.S3{}
	if _, err := s.Save(context.Background(), "application/x-bad", strings.NewReader("x")); err == nil {
		t.Error("S3.Save should reject an unsupported content type early")
	}
}

func TestS3_NewAndDelete(t *testing.T) {
	// Missing config errors early (no network).
	if _, err := storage.NewS3(context.Background(), storage.S3Config{}); err == nil {
		t.Error("NewS3 without endpoint/bucket should error")
	}
	// Delete of a URL outside the bucket is a no-op (no network).
	if err := (&storage.S3{}).Delete(context.Background(), "https://other/host/x.jpg"); err != nil {
		t.Errorf("Delete non-matching URL = %v, want nil", err)
	}
}
