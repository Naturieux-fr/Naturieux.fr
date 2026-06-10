package taxref

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ImportStats summarizes an import run.
type ImportStats struct {
	Read     int // data rows read
	Imported int // rows inserted
	Skipped  int // rows filtered out (synonyms, unwanted ranks, malformed)
}

// allowedRanks are the TAXREF RANG values kept for the quiz pool ("ES" =
// espèce / species).
var allowedRanks = map[string]bool{
	"ES": true,
}

// requiredColumns must be present in the native TAXREF header.
var requiredColumns = []string{"CD_NOM", "CD_REF", "RANG", "LB_NOM"}

// Import loads the native INPN TAXREF file (TAXREFv18.txt: tab-separated,
// quoted fields, UTF-8, header on the first line) into the database. Only
// valid species (CD_NOM == CD_REF, RANG == "ES") are kept. The whole load
// runs in one transaction. Columns are mapped by header name, so the import
// survives column reordering between TAXREF versions.
func Import(db *sql.DB, r io.Reader) (ImportStats, error) {
	var stats ImportStats

	reader := csv.NewReader(r)
	reader.Comma = '\t'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return stats, fmt.Errorf("taxref import: reading header: %w", err)
	}
	col, err := mapColumns(header)
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
			 kingdom, class, ordre, family, genus, taxa_group, fr)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cd_nom) DO NOTHING`)
	if err != nil {
		return stats, fmt.Errorf("taxref import: prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, fmt.Errorf("taxref import: read: %w", err)
		}
		stats.Read++

		row, ok := col.parseRow(fields)
		if !ok {
			stats.Skipped++
			continue
		}

		if _, err := stmt.Exec(row.cdNom, row.cdRef, row.cdTaxSup, row.rang,
			row.scientificName, row.vernacularName, row.kingdom, row.class,
			row.ordre, row.family, row.genus, row.group, row.fr); err != nil {
			return stats, fmt.Errorf("taxref import: insert cd_nom %d: %w", row.cdNom, err)
		}
		stats.Imported++
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

// columns maps TAXREF field names to their position in a row.
type columns map[string]int

// mapColumns builds a name→index map from the header line.
func mapColumns(header []string) (columns, error) {
	col := make(columns)
	for i, name := range header {
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
	group          string
	fr             string
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

	cdNom, err := strconv.Atoi(get("CD_NOM"))
	if err != nil {
		return taxonRow{}, false
	}
	cdRef, err := strconv.Atoi(get("CD_REF"))
	if err != nil {
		return taxonRow{}, false
	}
	// Keep only valid taxa: a synonym points to a different accepted name.
	if cdNom != cdRef {
		return taxonRow{}, false
	}

	rang := get("RANG")
	if !allowedRanks[rang] {
		return taxonRow{}, false
	}

	name := get("LB_NOM")
	if name == "" {
		return taxonRow{}, false
	}

	cdTaxSup, _ := strconv.Atoi(get("CD_TAXSUP"))

	return taxonRow{
		cdNom:          cdNom,
		cdRef:          cdRef,
		cdTaxSup:       cdTaxSup,
		rang:           rang,
		scientificName: name,
		vernacularName: get("NOM_VERN"),
		kingdom:        get("REGNE"),
		class:          get("CLASSE"),
		ordre:          get("ORDRE"),
		family:         get("FAMILLE"),
		genus:          genusOf(name),
		group:          get("GROUP2_INPN"),
		fr:             get("FR"),
	}, true
}

// genusOf extracts the genus from a binomial scientific name (the first
// word). TAXREF's native file has no denormalized genus column.
func genusOf(scientificName string) string {
	if i := strings.IndexByte(scientificName, ' '); i > 0 {
		return scientificName[:i]
	}
	return scientificName
}
