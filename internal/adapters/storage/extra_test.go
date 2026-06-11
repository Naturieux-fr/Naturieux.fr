package storage_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
)

func TestLocal_SaveAudioAndDelete(t *testing.T) {
	s, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	saved, err := s.Save(context.Background(), "audio/mpeg", strings.NewReader("fake-mp3"))
	if err != nil {
		t.Fatalf("Save audio: %v", err)
	}
	if !strings.HasSuffix(saved.URL, ".mp3") {
		t.Errorf("audio URL %q should end in .mp3", saved.URL)
	}

	if err := s.Delete(context.Background(), saved.URL); err != nil {
		t.Errorf("Delete: %v", err)
	}
	// Deleting again (missing) must not error.
	if err := s.Delete(context.Background(), saved.URL); err != nil {
		t.Errorf("Delete missing: %v", err)
	}
}

func TestLocal_SaveRejectsUnknownType(t *testing.T) {
	s, _ := storage.NewLocal(t.TempDir())
	if _, err := s.Save(context.Background(), "application/x-evil", strings.NewReader("x")); err == nil {
		t.Error("Save should reject an unsupported content type")
	}
}

func TestExtensionHelpers(t *testing.T) {
	if ext, err := storage.ExtensionFor("image/png"); err != nil || ext != ".png" {
		t.Errorf("ExtensionFor png = %q,%v", ext, err)
	}
	if _, err := storage.ExtensionFor("audio/mpeg"); err == nil {
		t.Error("ExtensionFor should reject audio")
	}
	if ext, err := storage.AudioExtensionFor("audio/ogg"); err != nil || ext != ".ogg" {
		t.Errorf("AudioExtensionFor ogg = %q,%v", ext, err)
	}
	if _, err := storage.AudioExtensionFor("image/png"); err == nil {
		t.Error("AudioExtensionFor should reject images")
	}
}
