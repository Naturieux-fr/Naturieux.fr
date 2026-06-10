// Command importtaxref loads a TAXREF Darwin Core taxon file into the SQLite
// database used by the server.
//
// Usage:
//
//	importtaxref -file path/to/taxon.txt [-db naturieux.db] [-version v18.0]
//
// The taxon file is the `taxon.txt` extracted from the TAXREF Darwin Core
// archive (https://ipt.gbif.fr/archive.do?r=taxref, CC-BY 4.0).
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
)

func main() {
	var (
		filePath = flag.String("file", "", "path to the TAXREF taxon.txt file (required)")
		dbPath   = flag.String("db", "naturieux.db", "path to the SQLite database")
		version  = flag.String("version", "", "TAXREF version label to record (e.g. v18.0)")
	)
	flag.Parse()

	if *filePath == "" {
		log.Fatal("the -file flag is required (path to taxon.txt)")
	}

	db, err := sqlite.Open(*dbPath)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := taxref.EnsureSchema(db); err != nil {
		log.Fatalf("creating schema: %v", err)
	}

	f, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("opening taxon file: %v", err)
	}
	defer func() { _ = f.Close() }()

	log.Printf("Importing TAXREF from %s ...", *filePath)
	stats, err := taxref.Import(db, f)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	if *version != "" {
		if err := taxref.SetMeta(db, "version", *version); err != nil {
			log.Fatalf("recording version: %v", err)
		}
	}

	repo := taxref.NewRepository(db)
	count, _ := repo.CountSpecies(context.Background())
	log.Printf("Done: %d read, %d imported, %d skipped (%d species now in database)",
		stats.Read, stats.Imported, stats.Skipped, count)
}
