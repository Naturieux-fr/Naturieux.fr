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
	genus           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_taxref_class  ON taxref_species(class);
CREATE INDEX IF NOT EXISTS idx_taxref_ordre  ON taxref_species(ordre);
CREATE INDEX IF NOT EXISTS idx_taxref_family ON taxref_species(family);
CREATE INDEX IF NOT EXISTS idx_taxref_genus  ON taxref_species(genus);

-- Locally owned photos, linked to a taxon by its TAXREF id (cd_nom).
CREATE TABLE IF NOT EXISTS taxref_photos (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	cd_nom      INTEGER NOT NULL,
	url         TEXT NOT NULL,
	attribution TEXT NOT NULL DEFAULT '',
	license     TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (cd_nom) REFERENCES taxref_species(cd_nom)
);

CREATE INDEX IF NOT EXISTS idx_taxref_photos_cdnom ON taxref_photos(cd_nom);

-- Metadata (e.g. the imported TAXREF version).
CREATE TABLE IF NOT EXISTS taxref_meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// EnsureSchema creates the TAXREF tables and indexes if they do not exist.
func EnsureSchema(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("applying taxref schema: %w", err)
	}
	return nil
}
