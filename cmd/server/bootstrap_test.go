package main

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
)

func TestBootstrapData_NoEnvIsNoOp(t *testing.T) {
	t.Setenv("BOOTSTRAP_TAXREF_URL", "")
	t.Setenv("BOOTSTRAP_OCCURRENCES_URL", "")
	db, _ := sqlite.Open(":memory:")
	defer func() { _ = db.Close() }()
	bootstrapData(db) // returns immediately, no panic
}

func TestIsZipAndOccurrenceEmpty(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "p.txt")
	_ = os.WriteFile(plain, []byte("hello"), 0o600)
	if isZip(plain) {
		t.Error("plain text should not be a zip")
	}

	zpath := filepath.Join(dir, "a.zip")
	writeZip(t, zpath, "occurrence.txt", "data")
	if !isZip(zpath) {
		t.Error("zip should be detected")
	}

	db, _ := sqlite.Open(":memory:")
	defer func() { _ = db.Close() }()
	// species_months lives in the taxref schema; create it.
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS species_months (cd_nom INTEGER, month INTEGER)`)
	if !occurrenceEmpty(db) {
		t.Error("expected empty occurrence data")
	}
	_, _ = db.Exec(`INSERT INTO species_months (cd_nom, month) VALUES (1, 6)`)
	if occurrenceEmpty(db) {
		t.Error("expected non-empty occurrence data")
	}
}

func TestImportFromURL_PlainAndZip(t *testing.T) {
	plainSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("col1\tcol2\nx\ty\n"))
	}))
	defer plainSrv.Close()

	var got string
	if err := importFromURL(plainSrv.URL, "x", func(r io.Reader) error {
		b, _ := io.ReadAll(r)
		got = string(b)
		return nil
	}); err != nil {
		t.Fatalf("importFromURL plain: %v", err)
	}
	if got == "" {
		t.Error("importer received no data")
	}

	// Zip-served content.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, _ := zw.Create("occurrence.txt")
	_, _ = f.Write([]byte("zipped-data"))
	_ = zw.Close()
	zipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf.Bytes())
	}))
	defer zipSrv.Close()

	got = ""
	if err := importFromURL(zipSrv.URL, "occurrence", func(r io.Reader) error {
		b, _ := io.ReadAll(r)
		got = string(b)
		return nil
	}); err != nil {
		t.Fatalf("importFromURL zip: %v", err)
	}
	if got != "zipped-data" {
		t.Errorf("zip importer got %q", got)
	}

	// A bad URL errors.
	if err := importFromURL("http://127.0.0.1:0/nope", "x", func(io.Reader) error { return nil }); err == nil {
		t.Error("bad URL should error")
	}
}

func writeZip(t *testing.T, path, name, content string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, _ := zw.Create(name)
	_, _ = f.Write([]byte(content))
	_ = zw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write zip: %v", err)
	}
}
