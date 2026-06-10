package taxref

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// maxLineBytes is the scanner buffer ceiling; TAXREF rows are short but a few
// names with long authorship strings can exceed the default 64 KB token.
const maxLineBytes = 1024 * 1024

// ImportStats summarizes an import run.
type ImportStats struct {
	Read     int // data rows read
	Imported int // rows inserted
	Skipped  int // rows filtered out (synonyms, unwanted ranks, malformed)
}

// allowedRanks are the Darwin Core taxonRank values kept for the quiz pool.
var allowedRanks = map[string]bool{
	"species": true,
}

// Import loads a TAXREF Darwin Core taxon file (TSV, UTF-8, header on the
// first line) into the database. Only valid taxa (taxonID == acceptedName
// UsageID) of an allowed rank are kept. The whole load runs in one
// transaction for speed. Columns are mapped by header name, not position, so
// the import survives column reordering between TAXREF versions.
func Import(db *sql.DB, r io.Reader) (ImportStats, error) {
	var stats ImportStats

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	if !scanner.Scan() {
		return stats, fmt.Errorf("taxref import: empty input")
	}
	col, err := mapColumns(scanner.Text())
	if err != nil {
		return stats, err
	}

	tx, err := db.Begin()
	if err != nil {
		return stats, fmt.Errorf("taxref import: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO taxref_species
			(cd_nom, cd_ref, cd_taxsup, rang, scientific_name, vernacular_name,
			 kingdom, class, ordre, family, genus)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cd_nom) DO NOTHING`)
	if err != nil {
		return stats, fmt.Errorf("taxref import: prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		stats.Read++

		row, ok := col.parseRow(fields)
		if !ok {
			stats.Skipped++
			continue
		}

		if _, err := stmt.Exec(row.cdNom, row.cdRef, row.cdTaxSup, row.rang,
			row.scientificName, row.vernacularName, row.kingdom, row.class,
			row.ordre, row.family, row.genus); err != nil {
			return stats, fmt.Errorf("taxref import: insert cd_nom %d: %w", row.cdNom, err)
		}
		stats.Imported++
	}
	if err := scanner.Err(); err != nil {
		return stats, fmt.Errorf("taxref import: read: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return stats, fmt.Errorf("taxref import: commit: %w", err)
	}
	return stats, nil
}

// SetMeta stores a metadata key/value (e.g. the TAXREF version).
func SetMeta(db *sql.DB, key, value string) error {
	_, err := db.Exec(`
		INSERT INTO taxref_meta (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("taxref set meta: %w", err)
	}
	return nil
}

// columns maps Darwin Core field names to their position in a row.
type columns map[string]int

// requiredColumns must be present in the header for the import to proceed.
var requiredColumns = []string{
	"taxonID", "acceptedNameUsageID", "scientificName", "taxonRank",
}

// mapColumns builds a name→index map from the header line.
func mapColumns(header string) (columns, error) {
	col := make(columns)
	for i, name := range strings.Split(header, "\t") {
		col[strings.TrimSpace(name)] = i
	}
	for _, name := range requiredColumns {
		if _, ok := col[name]; !ok {
			return nil, fmt.Errorf("taxref import: missing column %q in header", name)
		}
	}
	return col, nil
}

// taxonRow is a parsed, filtered TAXREF row ready for insertion.
type taxonRow struct {
	cdNom          int
	cdRef          int
	cdTaxSup       int
	rang           string
	scientificName string
	vernacularName string
	kingdom        string
	class          string
	ordre          string
	family         string
	genus          string
}

// parseRow extracts and filters one data row. It returns ok=false for
// synonyms, unwanted ranks, or malformed rows.
func (c columns) parseRow(fields []string) (taxonRow, bool) {
	get := func(name string) string {
		i, ok := c[name]
		if !ok || i >= len(fields) {
			return ""
		}
		return strings.TrimSpace(fields[i])
	}

	cdNom, err := strconv.Atoi(get("taxonID"))
	if err != nil {
		return taxonRow{}, false
	}
	cdRef, err := strconv.Atoi(get("acceptedNameUsageID"))
	if err != nil {
		return taxonRow{}, false
	}
	// Keep only valid taxa: a synonym points to a different accepted name.
	if cdNom != cdRef {
		return taxonRow{}, false
	}

	rang := get("taxonRank")
	if !allowedRanks[rang] {
		return taxonRow{}, false
	}

	name := get("scientificName")
	if name == "" {
		return taxonRow{}, false
	}

	cdTaxSup, _ := strconv.Atoi(get("parentNameUsageID"))

	return taxonRow{
		cdNom:          cdNom,
		cdRef:          cdRef,
		cdTaxSup:       cdTaxSup,
		rang:           rang,
		scientificName: name,
		vernacularName: get("vernacularName"),
		kingdom:        get("kingdom"),
		class:          get("class"),
		ordre:          get("order"),
		family:         get("family"),
		genus:          get("genus"),
	}, true
}
