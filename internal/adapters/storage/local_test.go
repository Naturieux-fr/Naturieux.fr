package storage_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
)

func TestLocal_SaveServesUnderMedia(t *testing.T) {
	dir := t.TempDir()
	s, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	saved, err := s.Save(context.Background(), "image/jpeg", strings.NewReader("fake-jpeg-bytes"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !strings.HasPrefix(saved.URL, "/media/") || !strings.HasSuffix(saved.URL, ".jpg") {
		t.Errorf("URL = %s, want /media/<uuid>.jpg", saved.URL)
	}

	// The file exists on disk with the served name.
	name := strings.TrimPrefix(saved.URL, "/media/")
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("stored file unreadable: %v", err)
	}
	if string(content) != "fake-jpeg-bytes" {
		t.Errorf("stored content = %q, want the uploaded bytes", content)
	}
}

func TestLocal_Save_RejectsUnsupportedType(t *testing.T) {
	s, _ := storage.NewLocal(t.TempDir())
	if _, err := s.Save(context.Background(), "application/pdf", strings.NewReader("x")); !errors.Is(err, storage.ErrUnsupportedType) {
		t.Errorf("Save(pdf) error = %v, want ErrUnsupportedType", err)
	}
}

func TestLocal_Delete(t *testing.T) {
	dir := t.TempDir()
	s, _ := storage.NewLocal(dir)

	saved, _ := s.Save(context.Background(), "image/png", strings.NewReader("png"))
	if err := s.Delete(context.Background(), saved.URL); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	name := strings.TrimPrefix(saved.URL, "/media/")
	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Error("file should be gone after Delete()")
	}

	// Deleting external or unknown URLs is a harmless no-op.
	if err := s.Delete(context.Background(), "https://external.example/x.jpg"); err != nil {
		t.Errorf("Delete(external) error = %v, want nil", err)
	}
	if err := s.Delete(context.Background(), "/media/../../etc/passwd"); err != nil {
		t.Errorf("Delete(traversal) error = %v, want nil (ignored)", err)
	}
}

func TestExtensionFor(t *testing.T) {
	if ext, _ := storage.ExtensionFor("image/jpeg"); ext != ".jpg" {
		t.Errorf("jpeg ext = %s, want .jpg", ext)
	}
	if _, err := storage.ExtensionFor("text/plain"); !errors.Is(err, storage.ErrUnsupportedType) {
		t.Errorf("text/plain err = %v, want ErrUnsupportedType", err)
	}
}
