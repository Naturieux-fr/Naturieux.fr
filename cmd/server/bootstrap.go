package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
)

// bootstrapData optionally seeds the database from downloaded open-data files,
// once, on first run. Controlled by env vars:
//
//	BOOTSTRAP_TAXREF_URL       — TAXREF native file (.txt) or its .zip archive
//	BOOTSTRAP_OCCURRENCES_URL  — occurrence export for the lieu/saison filters
//
// This is a one-time setup download (like running the import tools by hand);
// there is NO per-request external call. Photos and sounds remain curator-added
// and are never downloaded.
func bootstrapData(db *sql.DB) {
	taxrefURL := strings.TrimSpace(os.Getenv("BOOTSTRAP_TAXREF_URL"))
	occURL := strings.TrimSpace(os.Getenv("BOOTSTRAP_OCCURRENCES_URL"))
	if taxrefURL == "" && occURL == "" {
		return
	}
	if err := taxref.EnsureSchema(db); err != nil {
		log.Printf("bootstrap: schema: %v", err)
		return
	}
	repo := taxref.NewRepository(db)
	ctx := context.Background()

	if taxrefURL != "" {
		if n, _ := repo.CountSpecies(ctx); n > 0 {
			log.Printf("Bootstrap: TAXREF already present (%d species), skipping download", n)
		} else {
			log.Printf("Bootstrap: downloading TAXREF reference…")
			if err := importFromURL(taxrefURL, "TAXREFv", func(r io.Reader) error {
				_, e := taxref.Import(db, r)
				return e
			}); err != nil {
				log.Printf("bootstrap taxref: %v", err)
			} else {
				_ = sqlite.Optimize(db)
				n, _ := repo.CountSpecies(ctx)
				log.Printf("Bootstrap: TAXREF imported (%d species)", n)
			}
		}
	}

	if occURL != "" {
		if occurrenceEmpty(db) {
			log.Printf("Bootstrap: downloading occurrences for lieu/saison (this file can be large)…")
			if err := importFromURL(occURL, "occurrence", func(r io.Reader) error {
				_, e := taxref.ImportOccurrences(db, r)
				return e
			}); err != nil {
				log.Printf("bootstrap occurrences: %v", err)
			} else {
				_ = sqlite.Optimize(db)
				log.Printf("Bootstrap: occurrences imported")
			}
		} else {
			log.Printf("Bootstrap: occurrence data already present, skipping download")
		}
	}
}

// occurrenceEmpty reports whether no lieu/saison data has been imported yet.
func occurrenceEmpty(db *sql.DB) bool {
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM species_months").Scan(&n)
	return n == 0
}

// importFromURL downloads url to a temp file and feeds the relevant content to
// importer. If the download is a ZIP, the entry whose name contains prefer (or,
// failing that, the largest .txt/.csv) is used.
func importFromURL(url, prefer string, importer func(io.Reader) error) error {
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url) // #nosec G704 -- operator-provided bootstrap URL (open data)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "naturieux-bootstrap-*")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("saving download: %w", err)
	}
	_ = tmp.Close()

	if isZip(tmp.Name()) {
		return importFromZip(tmp.Name(), prefer, importer)
	}
	f, err := os.Open(tmp.Name()) // #nosec G304 -- temp file we just wrote
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return importer(f)
}

// isZip reports whether the file starts with the ZIP magic number.
func isZip(path string) bool {
	f, err := os.Open(path) // #nosec G304 -- temp file we just wrote
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, 4)
	if _, err := io.ReadFull(f, head); err != nil {
		return false
	}
	return head[0] == 'P' && head[1] == 'K' && head[2] == 3 && head[3] == 4
}

// importFromZip opens the best data entry inside a ZIP and feeds it to importer.
func importFromZip(path, prefer string, importer func(io.Reader) error) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer func() { _ = zr.Close() }()

	var best *zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.ToLower(f.Name)
		isData := strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".csv")
		if !isData {
			continue
		}
		if prefer != "" && strings.Contains(name, strings.ToLower(prefer)) {
			best = f
			break
		}
		if best == nil || f.UncompressedSize64 > best.UncompressedSize64 {
			best = f
		}
	}
	if best == nil {
		return fmt.Errorf("no .txt/.csv data file found in archive")
	}
	rc, err := best.Open()
	if err != nil {
		return fmt.Errorf("reading %s: %w", best.Name, err)
	}
	defer func() { _ = rc.Close() }()
	return importer(rc)
}
