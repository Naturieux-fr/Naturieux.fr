// Command importphotos bulk-loads a photo collection into the TAXREF photo
// table from a mapping CSV.
//
// The CSV is semicolon-separated with a header:
//
//	photo;groupe_taxonomique;nom_scientifique
//	P1000733.JPG;Amphibiens;Bombina variegata
//
// Each row's scientific name is resolved to its TAXREF cd_nom, the image file
// (looked up in -dir) is stored via the configured backend (STORAGE env:
// local or s3), and a photo record is created. Non-image files (e.g. RAW
// .RW2) and unmatched names are skipped and reported.
//
// Usage:
//
//	importphotos -csv BDD_test.csv -dir ./photos [-db naturieux.db]
//	             [-attribution "(c) ..."] [-license cc-by] [-difficulty beginner]
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/media"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

func main() {
	var (
		csvPath     = flag.String("csv", "", "mapping CSV (photo;groupe;nom_scientifique) (required)")
		dir         = flag.String("dir", ".", "directory holding the photo files")
		dbPath      = flag.String("db", "naturieux.db", "SQLite database path")
		attribution = flag.String("attribution", "", "attribution applied to every imported photo")
		license     = flag.String("license", "", "license code applied to every imported photo (e.g. cc-by)")
		difficulty  = flag.String("difficulty", "", "difficulty applied to every imported photo (beginner..master)")
		dryRun      = flag.Bool("dry-run", false, "resolve and validate without storing anything")
	)
	flag.Parse()

	if *csvPath == "" {
		log.Fatal("the -csv flag is required")
	}

	rows, err := taxref.ParsePhotoCSV(*csvPath)
	if err != nil {
		log.Fatalf("reading CSV: %v", err)
	}
	log.Printf("CSV: %d rows", len(rows))

	db, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := taxref.EnsureSchema(db); err != nil {
		log.Fatalf("schema: %v", err)
	}
	repo := taxref.NewRepository(db)

	store, err := storage.FromEnv(context.Background())
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	ctx := context.Background()
	var imported, converted, unmatched, missing, skipped int

	for _, row := range rows {
		cdNom, err := repo.CdNomByScientificName(ctx, row.ScientificName)
		if errors.Is(err, ports.ErrNotFound) {
			log.Printf("  ✗ nom introuvable dans TAXREF: %q (%s)", row.ScientificName, row.Photo)
			unmatched++
			continue
		}
		if err != nil {
			log.Fatalf("lookup %q: %v", row.ScientificName, err)
		}

		path := filepath.Join(*dir, row.Photo)
		raw, err := os.ReadFile(path)
		if err != nil {
			log.Printf("  ✗ fichier absent: %s", row.Photo)
			missing++
			continue
		}

		contentType, body, fromRaw, ok := prepareImage(row.Photo, raw)
		if !ok {
			log.Printf("  ⊘ ignoré (format non pris en charge): %s", row.Photo)
			skipped++
			continue
		}
		if fromRaw {
			log.Printf("  ⟳ RAW converti en JPEG: %s", row.Photo)
			converted++
		}

		if *dryRun {
			imported++
			continue
		}

		saved, err := store.Save(ctx, contentType, bytes.NewReader(body))
		if err != nil {
			log.Fatalf("storing %s: %v", row.Photo, err)
		}
		if _, err := repo.AddPhoto(ctx, cdNom, saved.URL, *attribution, *license, *difficulty); err != nil {
			log.Fatalf("recording %s: %v", row.Photo, err)
		}
		imported++
	}

	verb := "imported"
	if *dryRun {
		verb = "would import"
	}
	log.Printf("Done: %d %s (%d converted from RAW), %d unmatched names, %d missing files, %d skipped",
		imported, verb, converted, unmatched, missing, skipped)
}

// prepareImage returns the content type and bytes to store for a photo. Web
// images are passed through; RAW files are converted to their embedded JPEG
// preview. fromRaw reports whether a conversion happened.
func prepareImage(name string, data []byte) (contentType string, body []byte, fromRaw bool, ok bool) {
	ct := http.DetectContentType(data)
	if _, err := storage.ExtensionFor(ct); err == nil {
		return ct, data, false, true
	}

	ext := strings.ToLower(filepath.Ext(name))
	if media.IsRawExtension(ext) {
		if jpg, err := media.ExtractPreviewJPEG(data); err == nil {
			return "image/jpeg", jpg, true, true
		}
	}
	return "", nil, false, false
}
