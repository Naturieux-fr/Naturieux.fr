package taxref

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
)

func TestImportOccurrences_AggregatesPresence(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	// Two reference species to match against.
	_, _ = db.Exec(`INSERT INTO taxref_species (cd_nom, cd_ref, rang, scientific_name) VALUES
		(351, 351, 'ES', 'Rana temporaria'),
		(60585, 60585, 'ES', 'Vulpes vulpes')`)

	// Synthetic GBIF-style export. Rana: month 3 ×2 + Île-de-France ×3 (kept);
	// month 7 ×1 (dropped, below minMonthHits). Vulpes: a non-FR row is ignored.
	csv := strings.Join([]string{
		"species\tmonth\tlevel1Name\tcountryCode",
		"Rana temporaria Linnaeus, 1758\t3\tÎle-de-France\tFR",
		"Rana temporaria\t3\tÎle-de-France\tFR",
		"Rana temporaria\t7\tÎle-de-France\tFR",
		"Rana temporaria\t3\tÎle-de-France\tFR",
		"Vulpes vulpes\t5\tOccitanie\tFR",
		"Vulpes vulpes\t5\tBretagne\tDE",
	}, "\n")

	stats, err := ImportOccurrences(db, strings.NewReader(csv))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if stats.Matched != 5 { // 5 FR/known rows matched (the DE row is skipped before matching)
		t.Errorf("matched = %d, want 5", stats.Matched)
	}

	// Rana is present in March (3 hits ≥ 2) but not July (1 hit < 2).
	if !hasMonth(t, db, 351, 3) {
		t.Error("Rana should be present in March")
	}
	if hasMonth(t, db, 351, 7) {
		t.Error("Rana should NOT be present in July (below threshold)")
	}
	// Île-de-France region kept (3 hits ≥ 3).
	if !hasRegion(t, db, 351, "Île-de-France") {
		t.Error("Rana should be present in Île-de-France")
	}
	// Vulpes Occitanie had only 1 hit (< 3) → not kept.
	if hasRegion(t, db, 60585, "Occitanie") {
		t.Error("Vulpes Occitanie should be below threshold")
	}
}

func TestImportOccurrences_MatchesByID(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	_, _ = db.Exec(`INSERT INTO taxref_species (cd_nom, cd_ref, rang, scientific_name) VALUES
		(60585, 60585, 'ES', 'Vulpes vulpes')`)

	// INPN-style export keyed by CD_REF (the TAXREF id) — no name column needed.
	csv := strings.Join([]string{
		"cd_ref\tmois\tnom_region",
		"60585\t11\tOccitanie",
		"60585\t11\tOccitanie",
		"60585\t11\tOccitanie",
		"99999\t11\tOccitanie", // unknown id → ignored
	}, "\n")

	stats, err := ImportOccurrences(db, strings.NewReader(csv))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if stats.Matched != 3 {
		t.Errorf("matched = %d, want 3 (id 99999 unknown)", stats.Matched)
	}
	if !hasMonth(t, db, 60585, 11) {
		t.Error("Vulpes should be present in November via id match")
	}
	if !hasRegion(t, db, 60585, "Occitanie") {
		t.Error("Vulpes should be present in Occitanie via id match")
	}
}

func hasMonth(t *testing.T, db *sql.DB, cd, m int) bool {
	t.Helper()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM species_months WHERE cd_nom = ? AND month = ?`, cd, m).Scan(&n)
	return n > 0
}

func hasRegion(t *testing.T, db *sql.DB, cd int, region string) bool {
	t.Helper()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM species_regions WHERE cd_nom = ? AND region = ?`, cd, region).Scan(&n)
	return n > 0
}
