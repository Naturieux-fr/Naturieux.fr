// Command importtaxref loads the native INPN TAXREF file into the SQLite
// database used by the server.
//
// Usage:
//
//	importtaxref -file path/to/TAXREFv18.txt [-db naturieux.db] [-version v18.0]
//
// The file is TAXREFvNN.txt from the native TAXREF archive (INPN, License
// Ouverte). The native file carries the GROUP1/2/3_INPN columns used for the
// quiz categories (Mammifères, Oiseaux, Reptiles, Amphibiens…) and the FR
// metropolitan-presence column — unlike the reduced GBIF Darwin Core export.
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

	log.Printf("Compacting database…")
	if err := sqlite.Optimize(db); err != nil {
		log.Printf("warning: optimize failed: %v", err)
	}

	repo := taxref.NewRepository(db)
	count, _ := repo.CountSpecies(context.Background())
	log.Printf("Done: %d read, %d imported, %d skipped (%d species now in database)",
		stats.Read, stats.Imported, stats.Skipped, count)
}
