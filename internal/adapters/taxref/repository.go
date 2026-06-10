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

// categoryRule maps a quiz category to a TAXREF column + value. Animal groups
// use the INPN group (GROUP2_INPN), which cleanly separates Reptiles,
// Amphibiens, etc. — unlike `class`, which TAXREF leaves empty for reptiles.
// Plants and fungi use the kingdom (REGNE).
type categoryRule struct{ column, value string }

// categoryRules accepts both the frontend's English iconic-taxon values and
// the French INPN group labels (so a species' own group round-trips through
// the distractor fallback).
var categoryRules = map[string]categoryRule{
	// English (frontend categories) → INPN group / kingdom
	"Mammalia": {"taxa_group", "Mammifères"},
	"Aves":     {"taxa_group", "Oiseaux"},
	"Reptilia": {"taxa_group", "Reptiles"},
	"Amphibia": {"taxa_group", "Amphibiens"},
	"Insecta":  {"taxa_group", "Insectes"},
	"Plantae":  {"kingdom", "Plantae"},
	"Fungi":    {"kingdom", "Fungi"},
	// French INPN group labels (a species' iconic taxon is its group)
	"Mammifères": {"taxa_group", "Mammifères"},
	"Oiseaux":    {"taxa_group", "Oiseaux"},
	"Reptiles":   {"taxa_group", "Reptiles"},
	"Amphibiens": {"taxa_group", "Amphibiens"},
	"Insectes":   {"taxa_group", "Insectes"},
	"Poissons":   {"taxa_group", "Poissons"},
}

// categoryFilter maps a UI category to the TAXREF column and value to filter
// on. An empty or unknown category matches everything.
func categoryFilter(category string) (column, value string) {
	if rule, ok := categoryRules[category]; ok {
		return rule.column, rule.value
	}
	return "", ""
}

// iconicTaxonOf derives the app's iconic-taxon label: the INPN group when
// known, otherwise the kingdom.
func iconicTaxonOf(group, kingdom string) string {
	if group != "" {
		return group
	}
	return kingdom
}

// GetByID retrieves a species (with its photos) by its TAXREF id.
func (r *Repository) GetByID(ctx context.Context, id int) (*species.Species, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+speciesColumns+` FROM taxref_species WHERE cd_nom = ?`, id)

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
//
// With HasPhotos the query is driven from the (small) owned-photo table and
// joined to the reference, rather than scanning the ~200k-row reference and
// testing photo existence per row — a ~300x speedup measured on TAXREF v18.
func (r *Repository) GetRandom(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 1
	}

	// Prefer photos at the requested difficulty, but fall back to any so a
	// session never runs dry when a difficulty has too few photos.
	speciesList, err := r.randomSpecies(ctx, filter, filter.Difficulty, limit, nil)
	if err != nil {
		return nil, err
	}
	if filter.HasPhotos && filter.Difficulty != "" && len(speciesList) < limit {
		exclude := append(append([]int{}, filter.ExcludeIDs...), idsOf(speciesList)...)
		more, err := r.randomSpecies(ctx, filter, "", limit-len(speciesList), exclude)
		if err != nil {
			return nil, err
		}
		speciesList = append(speciesList, more...)
	}

	for _, sp := range speciesList {
		if err := r.attachPhotos(ctx, sp); err != nil {
			return nil, err
		}
	}
	return speciesList, nil
}

// randomSpecies runs one random-selection query. When the filter requires
// photos the query is driven from the (small) photo table; difficulty, when
// non-empty, restricts to photos at that level. extraExclude augments the
// filter's excluded ids.
func (r *Repository) randomSpecies(ctx context.Context, filter ports.SpeciesFilter, difficulty string, limit int, extraExclude []int) ([]*species.Species, error) {
	cols := "s." + strings.ReplaceAll(speciesColumns, ", ", ", s.")

	var query string
	var where []string
	var args []interface{}

	if filter.HasPhotos {
		query = `SELECT ` + cols + ` FROM taxref_photos p JOIN taxref_species s ON s.cd_nom = p.cd_nom`
		if difficulty != "" {
			where = append(where, "p.difficulty = ?")
			args = append(args, difficulty)
		}
	} else {
		query = `SELECT ` + cols + ` FROM taxref_species s`
	}

	if col, val := categoryFilter(filter.IconicTaxon); col != "" {
		where = append(where, "s."+col+" = ?")
		args = append(args, val)
	}
	exclude := append(append([]int{}, filter.ExcludeIDs...), extraExclude...)
	if len(exclude) > 0 {
		where = append(where, "s.cd_nom NOT IN ("+placeholders(len(exclude))+")")
		for _, id := range exclude {
			args = append(args, id)
		}
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	if filter.HasPhotos {
		// A species may own several photos; collapse to one row each.
		query += " GROUP BY s.cd_nom"
	}
	query += " ORDER BY RANDOM() LIMIT ?"
	args = append(args, limit)

	return r.querySpecies(ctx, query, args...)
}

// idsOf returns the cd_nom of each species.
func idsOf(list []*species.Species) []int {
	ids := make([]int, len(list))
	for i, sp := range list {
		ids[i] = sp.ID()
	}
	return ids
}

// GetSimilar returns species taxonomically close to the given one, ranked by
// proximity (same genus, then same family, then same order) so the quiz can
// build hard, plausible distractors. Closeness is read straight from the
// denormalized rank columns — no tree traversal. Distractors only need a
// name, so photos are not required here.
func (r *Repository) GetSimilar(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
	var genus, family, ordre string
	err := r.db.QueryRowContext(ctx,
		`SELECT genus, family, ordre FROM taxref_species WHERE cd_nom = ?`, speciesID).Scan(&genus, &family, &ordre)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("taxref get similar: %w", err)
	}

	// Single query: candidates sharing the genus, family or order, ordered
	// closest-first and randomized within each proximity tier. Empty rank
	// values can never match a real row, so they are harmless.
	return r.querySpecies(ctx, `
		SELECT `+speciesColumns+`
		FROM taxref_species
		WHERE cd_nom != ?
		  AND (genus = ? OR family = ? OR ordre = ?)
		ORDER BY
			CASE
				WHEN genus = ?  THEN 1
				WHEN family = ? THEN 2
				ELSE 3
			END,
			CASE WHEN fr IN ('P', 'E', 'S', 'I') THEN 0 ELSE 1 END,
			RANDOM()
		LIMIT ?`,
		speciesID, nonEmpty(genus), nonEmpty(family), nonEmpty(ordre),
		nonEmpty(genus), nonEmpty(family), limit)
}

// nonEmpty maps an empty rank to a sentinel that cannot match any real row,
// so an absent genus/family/order simply contributes no candidates.
func nonEmpty(s string) string {
	if s == "" {
		return "\x00"
	}
	return s
}

// Search finds species whose scientific or vernacular name matches the query.
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]*species.Species, error) {
	if limit <= 0 {
		limit = 10
	}
	like := "%" + query + "%"
	return r.querySpecies(ctx,
		`SELECT `+speciesColumns+` FROM taxref_species
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

// speciesColumns is the column list scanned by scanSpecies.
const speciesColumns = "cd_nom, scientific_name, vernacular_name, taxa_group, kingdom"

// scanSpecies reads the standard species columns and builds a domain Species.
func (r *Repository) scanSpecies(row rowScanner) (*species.Species, error) {
	var (
		cdNom          int
		scientific     string
		vernacular     string
		group, kingdom string
	)
	if err := row.Scan(&cdNom, &scientific, &vernacular, &group, &kingdom); err != nil {
		return nil, err
	}

	sp, err := species.New(cdNom, scientific, firstVernacular(vernacular), iconicTaxonOf(group, kingdom))
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

// PhotoRecord is an owned photo as managed by the back-office.
type PhotoRecord struct {
	ID          int    `json:"id"`
	CdNom       int    `json:"cd_nom"`
	URL         string `json:"url"`
	Attribution string `json:"attribution"`
	License     string `json:"license"`
	Difficulty  string `json:"difficulty"`
}

// AddPhoto inserts a locally owned photo for a taxon and returns its id.
// The taxon must exist in the reference.
func (r *Repository) AddPhoto(ctx context.Context, cdNom int, url, attribution, license, difficulty string) (int, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM taxref_species WHERE cd_nom = ?`, cdNom).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ports.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("taxref add photo: %w", err)
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO taxref_photos (cd_nom, url, attribution, license, difficulty)
		VALUES (?, ?, ?, ?, ?)`, cdNom, url, attribution, license, difficulty)
	if err != nil {
		return 0, fmt.Errorf("taxref add photo: %w", err)
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// ListPhotos returns the owned photos for a taxon.
func (r *Repository) ListPhotos(ctx context.Context, cdNom int) ([]PhotoRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, cd_nom, url, attribution, license, difficulty
		FROM taxref_photos WHERE cd_nom = ? ORDER BY id`, cdNom)
	if err != nil {
		return nil, fmt.Errorf("taxref list photos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	photos := make([]PhotoRecord, 0)
	for rows.Next() {
		var p PhotoRecord
		if err := rows.Scan(&p.ID, &p.CdNom, &p.URL, &p.Attribution, &p.License, &p.Difficulty); err != nil {
			return nil, fmt.Errorf("scanning photo: %w", err)
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

// DeletePhoto removes an owned photo by its id and returns its URL, so the
// caller can also remove the stored file.
func (r *Repository) DeletePhoto(ctx context.Context, id int) (string, error) {
	var url string
	err := r.db.QueryRowContext(ctx, `SELECT url FROM taxref_photos WHERE id = ?`, id).Scan(&url)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ports.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("taxref delete photo: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM taxref_photos WHERE id = ?`, id); err != nil {
		return "", fmt.Errorf("taxref delete photo: %w", err)
	}
	return url, nil
}

// CdNomByScientificName resolves an exact scientific name to its cd_nom.
// Returns ErrNotFound when no valid species matches.
func (r *Repository) CdNomByScientificName(ctx context.Context, name string) (int, error) {
	var cdNom int
	err := r.db.QueryRowContext(ctx,
		`SELECT cd_nom FROM taxref_species WHERE scientific_name = ? LIMIT 1`, name).Scan(&cdNom)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ports.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("taxref lookup %q: %w", name, err)
	}
	return cdNom, nil
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
