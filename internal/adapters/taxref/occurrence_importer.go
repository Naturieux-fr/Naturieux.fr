package taxref

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// OccurrenceStats summarizes an occurrence import run.
type OccurrenceStats struct {
	Read    int // data rows read
	Matched int // rows matched to a TAXREF species
	Species int // distinct species written
}

// occurrence import keeps a presence only when a (species, month) or
// (species, region) pair is seen at least this many times, to drop stray
// records and mislocated observations.
const (
	minMonthHits  = 2
	minRegionHits = 3
)

// ImportOccurrences aggregates a downloaded occurrence export (GBIF "simple
// CSV"/DwC-A or INPN, tab-separated with a header) into the species_months and
// species_regions tables. It maps columns by header name so it tolerates the
// different export layouts:
//   - species name: "species" (binomial), else "scientific_name"/"nom_valide"
//   - month:        "month", else "mois"
//   - region:       "level1Name" (GADM région), else "stateProvince"/"nom_region"
//   - country:      "countryCode" (rows kept only if FR, when present)
//
// Species are matched to TAXREF by scientific name. There is no runtime API:
// this runs once, like the TAXREF import.
func ImportOccurrences(db *sql.DB, r io.Reader) (OccurrenceStats, error) {
	var stats OccurrenceStats

	nameToCd, err := loadNameIndex(db)
	if err != nil {
		return stats, err
	}

	reader := csv.NewReader(r)
	reader.Comma = '\t'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	header, err := reader.Read()
	if err != nil {
		return stats, fmt.Errorf("occurrence import: reading header: %w", err)
	}
	idx := indexOccurrenceColumns(header)
	if idx.species < 0 {
		return stats, fmt.Errorf("occurrence import: no species/scientific-name column found")
	}

	// Aggregate in memory, keyed by cd_nom (only matched species).
	months := map[int]map[int]int{}     // cd_nom -> month -> count
	regions := map[int]map[string]int{} // cd_nom -> region -> count

	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, fmt.Errorf("occurrence import: read: %w", err)
		}
		stats.Read++

		if idx.country >= 0 && idx.country < len(fields) {
			if c := strings.ToUpper(strings.TrimSpace(fields[idx.country])); c != "" && c != "FR" && c != "FRA" {
				continue
			}
		}
		name := normalizeBinomial(get(fields, idx.species))
		cd, ok := nameToCd[name]
		if !ok {
			continue
		}
		stats.Matched++

		if m := monthOf(get(fields, idx.month)); m >= 1 && m <= 12 {
			if months[cd] == nil {
				months[cd] = map[int]int{}
			}
			months[cd][m]++
		}
		if region := strings.TrimSpace(get(fields, idx.region)); region != "" {
			if regions[cd] == nil {
				regions[cd] = map[string]int{}
			}
			regions[cd][region]++
		}
	}

	if err := writeOccurrences(db, months, regions, &stats); err != nil {
		return stats, err
	}
	return stats, nil
}

// loadNameIndex maps every TAXREF binomial scientific name to its cd_nom.
func loadNameIndex(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`SELECT cd_nom, scientific_name FROM taxref_species`)
	if err != nil {
		return nil, fmt.Errorf("occurrence import: loading names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]int, 1<<18)
	for rows.Next() {
		var cd int
		var name string
		if err := rows.Scan(&cd, &name); err != nil {
			return nil, err
		}
		if key := normalizeBinomial(name); key != "" {
			if _, exists := out[key]; !exists {
				out[key] = cd
			}
		}
	}
	return out, rows.Err()
}

func writeOccurrences(db *sql.DB, months map[int]map[int]int, regions map[int]map[string]int, stats *OccurrenceStats) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("occurrence import: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM species_months`); err != nil {
		return fmt.Errorf("occurrence import: clearing months: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM species_regions`); err != nil {
		return fmt.Errorf("occurrence import: clearing regions: %w", err)
	}

	mStmt, err := tx.Prepare(`INSERT OR IGNORE INTO species_months (cd_nom, month) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = mStmt.Close() }()
	rStmt, err := tx.Prepare(`INSERT OR IGNORE INTO species_regions (cd_nom, region) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = rStmt.Close() }()

	written := map[int]bool{}
	for cd, ms := range months {
		for m, n := range ms {
			if n >= minMonthHits {
				if _, err := mStmt.Exec(cd, m); err != nil {
					return err
				}
				written[cd] = true
			}
		}
	}
	for cd, rs := range regions {
		for region, n := range rs {
			if n >= minRegionHits {
				if _, err := rStmt.Exec(cd, region); err != nil {
					return err
				}
				written[cd] = true
			}
		}
	}
	stats.Species = len(written)
	return tx.Commit()
}

// occurrenceColumns holds the resolved column indices.
type occurrenceColumns struct {
	species int
	month   int
	region  int
	country int
}

func indexOccurrenceColumns(header []string) occurrenceColumns {
	idx := occurrenceColumns{species: -1, month: -1, region: -1, country: -1}
	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "species":
			idx.species = i
		case "scientific_name", "scientificname", "nom_valide", "nomvalide":
			if idx.species < 0 {
				idx.species = i
			}
		case "month", "mois":
			idx.month = i
		case "level1name":
			idx.region = i
		case "stateprovince", "nom_region", "region", "nomregion":
			if idx.region < 0 {
				idx.region = i
			}
		case "countrycode", "country":
			idx.country = i
		}
	}
	return idx
}

func get(fields []string, i int) string {
	if i < 0 || i >= len(fields) {
		return ""
	}
	return fields[i]
}

// normalizeBinomial lowercases and keeps the first two words (genus + species),
// so "Rana temporaria Linnaeus, 1758" matches "Rana temporaria".
func normalizeBinomial(name string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(name)))
	if len(parts) < 2 {
		return strings.Join(parts, " ")
	}
	return parts[0] + " " + parts[1]
}

// monthOf parses a month either as a number (1-12) or from an ISO date prefix.
func monthOf(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	// e.g. "2021-03-14" -> 3
	if i := strings.IndexByte(v, '-'); i >= 0 {
		rest := v[i+1:]
		if j := strings.IndexByte(rest, '-'); j >= 0 {
			if n, err := strconv.Atoi(rest[:j]); err == nil {
				return n
			}
		}
	}
	return 0
}
