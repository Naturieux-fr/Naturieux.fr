package taxref

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// Repository is a ports.SpeciesRepository backed by TAXREF data and a locally
// owned photo collection, both stored in SQLite.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a TAXREF-backed species repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// classTaxa are the iconic-taxon categories that map to the TAXREF `class`
// column; the rest (Plantae, Fungi) map to `kingdom`.
var classTaxa = map[string]bool{
	"Mammalia": true, "Aves": true, "Reptilia": true,
	"Amphibia": true, "Insecta": true,
}

// categoryFilter maps a UI category to the TAXREF column and value to filter
// on. An empty category matches everything.
func categoryFilter(category string) (column, value string) {
	if category == "" {
		return "", ""
	}
	if classTaxa[category] {
		return "class", category
	}
	return "kingdom", category
}

// iconicTaxonOf derives the app's iconic-taxon label from TAXREF ranks.
func iconicTaxonOf(class, kingdom string) string {
	if classTaxa[class] {
		return class
	}
	if kingdom == "Plantae" || kingdom == "Fungi" {
		return kingdom
	}
	return class
}

// GetByID retrieves a species (with its photos) by its TAXREF id.
func (r *Repository) GetByID(ctx context.Context, id int) (*species.Species, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT cd_nom, scientific_name, vernacular_name, class, kingdom
		FROM taxref_species WHERE cd_nom = ?`, id)

	sp, err := r.scanSpecies(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.attachPhotos(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// GetRandom returns random species matching the filter. When HasPhotos is set
// (the case for the quiz's correct answer), only species that own at least
// one photo are returned, and their photos are attached.
func (r *Repository) GetRandom(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 1
	}

	var where []string
	var args []interface{}

	if col, val := categoryFilter(filter.IconicTaxon); col != "" {
		where = append(where, "s."+col+" = ?")
		args = append(args, val)
	}
	if filter.HasPhotos {
		where = append(where, "EXISTS (SELECT 1 FROM taxref_photos p WHERE p.cd_nom = s.cd_nom)")
	}
	if len(filter.ExcludeIDs) > 0 {
		where = append(where, "s.cd_nom NOT IN ("+placeholders(len(filter.ExcludeIDs))+")")
		for _, id := range filter.ExcludeIDs {
			args = append(args, id)
		}
	}

	query := `SELECT s.cd_nom, s.scientific_name, s.vernacular_name, s.class, s.kingdom
		FROM taxref_species s`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY RANDOM() LIMIT ?"
	args = append(args, limit)

	speciesList, err := r.querySpecies(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	for _, sp := range speciesList {
		if err := r.attachPhotos(ctx, sp); err != nil {
			return nil, err
		}
	}
	return speciesList, nil
}

// GetSimilar returns species taxonomically close to the given one (same genus
// first, then same family) to serve as plausible distractors. Distractors
// only need a name, so photos are not required here.
func (r *Repository) GetSimilar(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
	var genus, family string
	err := r.db.QueryRowContext(ctx,
		`SELECT genus, family FROM taxref_species WHERE cd_nom = ?`, speciesID).Scan(&genus, &family)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("taxref get similar: %w", err)
	}

	// Prefer same genus, fall back to same family to reach the limit.
	result := make([]*species.Species, 0, limit)
	seen := map[int]bool{speciesID: true}

	for _, clause := range []struct {
		column string
		value  string
	}{{"genus", genus}, {"family", family}} {
		if clause.value == "" || len(result) >= limit {
			continue
		}
		batch, err := r.querySpecies(ctx, `
			SELECT cd_nom, scientific_name, vernacular_name, class, kingdom
			FROM taxref_species
			WHERE `+clause.column+` = ? AND cd_nom != ?
			ORDER BY RANDOM() LIMIT ?`, clause.value, speciesID, limit*2)
		if err != nil {
			return nil, err
		}
		for _, sp := range batch {
			if seen[sp.ID()] {
				continue
			}
			seen[sp.ID()] = true
			result = append(result, sp)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// Search finds species whose scientific or vernacular name matches the query.
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]*species.Species, error) {
	if limit <= 0 {
		limit = 10
	}
	like := "%" + query + "%"
	return r.querySpecies(ctx, `
		SELECT cd_nom, scientific_name, vernacular_name, class, kingdom
		FROM taxref_species
		WHERE scientific_name LIKE ? OR vernacular_name LIKE ?
		ORDER BY scientific_name LIMIT ?`, like, like, limit)
}

// querySpecies runs a query selecting the standard species columns.
func (r *Repository) querySpecies(ctx context.Context, query string, args ...interface{}) ([]*species.Species, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("taxref query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]*species.Species, 0)
	for rows.Next() {
		sp, err := r.scanSpecies(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sp)
	}
	return result, rows.Err()
}

// rowScanner abstracts *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanSpecies reads the standard species columns and builds a domain Species.
func (r *Repository) scanSpecies(row rowScanner) (*species.Species, error) {
	var (
		cdNom          int
		scientific     string
		vernacular     string
		class, kingdom string
	)
	if err := row.Scan(&cdNom, &scientific, &vernacular, &class, &kingdom); err != nil {
		return nil, err
	}

	sp, err := species.New(cdNom, scientific, firstVernacular(vernacular), iconicTaxonOf(class, kingdom))
	if err != nil {
		return nil, fmt.Errorf("taxref build species %d: %w", cdNom, err)
	}
	sp.SetRank("species")
	return sp, nil
}

// attachPhotos loads the owned photos for a species.
func (r *Repository) attachPhotos(ctx context.Context, sp *species.Species) error {
	rows, err := r.db.QueryContext(ctx,
		`SELECT url, attribution, license FROM taxref_photos WHERE cd_nom = ?`, sp.ID())
	if err != nil {
		return fmt.Errorf("taxref photos for %d: %w", sp.ID(), err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var url, attribution, license string
		if err := rows.Scan(&url, &attribution, &license); err != nil {
			return fmt.Errorf("scanning photo: %w", err)
		}
		sp.AddPhoto(species.Photo{
			URL:         url,
			MediumURL:   url,
			LargeURL:    url,
			Attribution: attribution,
			LicenseCode: license,
		})
	}
	return rows.Err()
}

// firstVernacular returns the first name from a comma-separated TAXREF
// vernacular field (e.g. "Renard roux, Renard, Goupil" → "Renard roux").
func firstVernacular(v string) string {
	if i := strings.IndexByte(v, ','); i >= 0 {
		return strings.TrimSpace(v[:i])
	}
	return strings.TrimSpace(v)
}

// placeholders returns n comma-separated SQL placeholders.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// AddPhoto inserts a locally owned photo for a taxon. Helper for tooling.
func (r *Repository) AddPhoto(ctx context.Context, cdNom int, url, attribution, license string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO taxref_photos (cd_nom, url, attribution, license)
		VALUES (?, ?, ?, ?)`, cdNom, url, attribution, license)
	if err != nil {
		return fmt.Errorf("taxref add photo: %w", err)
	}
	return nil
}

// CountSpecies returns how many taxa are loaded (useful for diagnostics).
func (r *Repository) CountSpecies(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM taxref_species`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// Version returns the imported TAXREF version, or "" if unknown.
func (r *Repository) Version(ctx context.Context) string {
	var v string
	_ = r.db.QueryRowContext(ctx, `SELECT value FROM taxref_meta WHERE key = 'version'`).Scan(&v)
	return v
}

// Ensure interface compliance.
var _ ports.SpeciesRepository = (*Repository)(nil)
