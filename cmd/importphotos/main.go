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
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
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
	var imported, unmatched, missing, skipped int

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
		f, err := os.Open(path)
		if err != nil {
			log.Printf("  ✗ fichier absent: %s", row.Photo)
			missing++
			continue
		}

		contentType, ok := detectImage(f)
		if !ok {
			log.Printf("  ⊘ ignoré (pas une image web, ex. RAW): %s", row.Photo)
			_ = f.Close()
			skipped++
			continue
		}

		if *dryRun {
			imported++
			_ = f.Close()
			continue
		}

		if _, err := f.Seek(0, 0); err != nil {
			log.Fatalf("seek %s: %v", row.Photo, err)
		}
		saved, err := store.Save(ctx, contentType, f)
		_ = f.Close()
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
	log.Printf("Done: %d %s, %d unmatched names, %d missing files, %d skipped (non-image)",
		imported, verb, unmatched, missing, skipped)
}

// detectImage sniffs the file's content type and reports whether it is a
// supported web image.
func detectImage(f *os.File) (string, bool) {
	head := make([]byte, 512)
	n, _ := f.Read(head)
	contentType := http.DetectContentType(head[:n])
	if _, err := storage.ExtensionFor(contentType); err != nil {
		return "", false
	}
	return contentType, true
}
