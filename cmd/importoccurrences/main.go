// Command importoccurrences aggregates a downloaded species-occurrence export
// (GBIF "simple CSV"/Darwin Core archive, or INPN) into the local database to
// power the lieu (region) and saison (month) quiz filters.
//
// This is a one-time import, like importtaxref — there is NO runtime API call.
// Obtain the file once from GBIF (https://www.gbif.org, filter Country=France,
// then "Download" → Simple) or INPN/OpenObs, unzip it, and run:
//
//	importoccurrences -file path/to/occurrence.txt [-db naturieux.db]
//
// The file is tab-separated with a header. Columns are mapped by name, so both
// the GBIF and INPN layouts work. TAXREF must already be imported (species are
// matched by scientific name).
package main

import (
	"flag"
	"log"
	"os"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
)

func main() {
	var (
		filePath = flag.String("file", "", "path to the occurrence export (tab-separated, required)")
		dbPath   = flag.String("db", "naturieux.db", "path to the SQLite database")
	)
	flag.Parse()

	if *filePath == "" {
		log.Fatal("the -file flag is required (path to the occurrence export)")
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
		log.Fatalf("opening occurrence file: %v", err)
	}
	defer func() { _ = f.Close() }()

	log.Printf("Importing occurrences from %s ...", *filePath)
	stats, err := taxref.ImportOccurrences(db, f)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}
	log.Printf("Compacting database…")
	if err := sqlite.Optimize(db); err != nil {
		log.Printf("warning: optimize failed: %v", err)
	}

	log.Printf("Done: %d rows read, %d matched to a species, %d species now have lieu/saison data",
		stats.Read, stats.Matched, stats.Species)
}
