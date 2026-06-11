// Package taxref provides a species repository backed by the French national
// taxonomic reference (TAXREF, published by PatriNat / MNHN) stored in SQLite,
// paired with a locally owned photo collection.
//
// TAXREF denormalizes the taxonomic ranks (kingdom, class, order, family,
// genus) as plain text on every row, so quiz queries — a random species in a
// group, distractors from the same family — are simple indexed lookups with
// no tree traversal.
package taxref

import (
	"database/sql"
	"fmt"
)

// schema defines the TAXREF species table and the locally owned photo table.
const schema = `
CREATE TABLE IF NOT EXISTS taxref_species (
	cd_nom          INTEGER PRIMARY KEY,
	cd_ref          INTEGER NOT NULL,
	cd_taxsup       INTEGER NOT NULL DEFAULT 0,
	rang            TEXT NOT NULL DEFAULT '',
	scientific_name TEXT NOT NULL,
	vernacular_name TEXT NOT NULL DEFAULT '',
	kingdom         TEXT NOT NULL DEFAULT '',
	class           TEXT NOT NULL DEFAULT '',
	ordre           TEXT NOT NULL DEFAULT '',
	family          TEXT NOT NULL DEFAULT '',
	genus           TEXT NOT NULL DEFAULT '',
	taxa_group      TEXT NOT NULL DEFAULT '',
	fr              TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_taxref_group  ON taxref_species(taxa_group);
CREATE INDEX IF NOT EXISTS idx_taxref_kingdom ON taxref_species(kingdom);
CREATE INDEX IF NOT EXISTS idx_taxref_family ON taxref_species(family);
CREATE INDEX IF NOT EXISTS idx_taxref_genus  ON taxref_species(genus);

-- Locally owned photos, linked to a taxon by its TAXREF id (cd_nom).
CREATE TABLE IF NOT EXISTS taxref_photos (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	cd_nom      INTEGER NOT NULL,
	url         TEXT NOT NULL,
	attribution TEXT NOT NULL DEFAULT '',
	license     TEXT NOT NULL DEFAULT '',
	difficulty  TEXT NOT NULL DEFAULT '',
	zones       TEXT NOT NULL DEFAULT '{}',
	FOREIGN KEY (cd_nom) REFERENCES taxref_species(cd_nom)
);

CREATE INDEX IF NOT EXISTS idx_taxref_photos_cdnom ON taxref_photos(cd_nom);

-- Locally owned recordings (bird songs, etc.) for the Sound quiz.
CREATE TABLE IF NOT EXISTS taxref_sounds (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	cd_nom      INTEGER NOT NULL,
	url         TEXT NOT NULL,
	attribution TEXT NOT NULL DEFAULT '',
	license     TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (cd_nom) REFERENCES taxref_species(cd_nom)
);

CREATE INDEX IF NOT EXISTS idx_taxref_sounds_cdnom ON taxref_sounds(cd_nom);

-- Aggregated occurrence presence per species, imported once from a downloaded
-- dataset (GBIF/INPN). No runtime API: these tables drive the lieu/saison
-- filters. A row means "this species is observed in this month / region".
CREATE TABLE IF NOT EXISTS species_months (
	cd_nom INTEGER NOT NULL,
	month  INTEGER NOT NULL,
	PRIMARY KEY (cd_nom, month)
);

CREATE TABLE IF NOT EXISTS species_regions (
	cd_nom INTEGER NOT NULL,
	region TEXT NOT NULL,
	PRIMARY KEY (cd_nom, region)
);

CREATE INDEX IF NOT EXISTS idx_species_months_m ON species_months(month);
CREATE INDEX IF NOT EXISTS idx_species_regions_r ON species_regions(region);

-- Metadata (e.g. the imported TAXREF version).
CREATE TABLE IF NOT EXISTS taxref_meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// EnsureSchema creates the TAXREF tables and indexes if they do not exist,
// and brings older databases up to date.
func EnsureSchema(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("applying taxref schema: %w", err)
	}
	// Add columns introduced after the initial schema, for older databases.
	for _, c := range []struct{ table, column, def string }{
		{"taxref_photos", "difficulty", "TEXT NOT NULL DEFAULT ''"},
		{"taxref_photos", "zones", "TEXT NOT NULL DEFAULT '{}'"},
		{"taxref_species", "taxa_group", "TEXT NOT NULL DEFAULT ''"},
		{"taxref_species", "fr", "TEXT NOT NULL DEFAULT ''"},
	} {
		has, err := columnExists(db, c.table, c.column)
		if err != nil {
			return err
		}
		if has {
			continue
		}
		if _, err := db.Exec("ALTER TABLE " + c.table + " ADD COLUMN " + c.column + " " + c.def); err != nil {
			return fmt.Errorf("migrating %s.%s: %w", c.table, c.column, err)
		}
	}
	return nil
}

// columnExists reports whether a table already has a column.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, fmt.Errorf("inspecting %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid, notnull, pk int
			name, ctype      string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
