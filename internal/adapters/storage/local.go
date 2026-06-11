package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// urlPrefix is the public path under which local files are served.
const urlPrefix = "/media/"

// Local stores files on the local filesystem and serves them under /media/.
type Local struct {
	dir string
}

// NewLocal creates a local storage rooted at dir, creating it if needed.
func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating media dir: %w", err)
	}
	return &Local{dir: dir}, nil
}

// Dir returns the storage directory (for wiring a file server).
func (l *Local) Dir() string {
	return l.dir
}

// Save writes the data to a uniquely named file and returns its public URL.
// The filename is generated (uuid + extension from the content type), never
// derived from user input, so it cannot escape the storage directory.
func (l *Local) Save(_ context.Context, contentType string, r io.Reader) (Saved, error) {
	ext, err := extensionForAny(contentType)
	if err != nil {
		return Saved{}, err
	}

	name := uuid.New().String() + ext
	path := filepath.Join(l.dir, name)

	f, err := os.Create(path)
	if err != nil {
		return Saved{}, fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, r); err != nil {
		_ = os.Remove(path)
		return Saved{}, fmt.Errorf("writing file: %w", err)
	}
	return Saved{URL: urlPrefix + name}, nil
}

// Delete removes a file addressed by its public URL. URLs that are not local
// media (e.g. external CC photos) are ignored.
func (l *Local) Delete(_ context.Context, url string) error {
	name, ok := strings.CutPrefix(url, urlPrefix)
	if !ok {
		return nil
	}
	// Guard against any path separators sneaking into the name.
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return nil
	}
	err := os.Remove(filepath.Join(l.dir, name))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file: %w", err)
	}
	return nil
}

// Ensure interface compliance.
var _ Storage = (*Local)(nil)
