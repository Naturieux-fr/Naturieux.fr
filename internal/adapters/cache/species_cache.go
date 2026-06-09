// Package cache provides a SQLite-backed caching layer around a species
// repository. It keeps quiz gameplay fast and respectful of the iNaturalist
// rate limits: reads are served from the local cache and only fall back to
// the wrapped source on a miss, while a background warmer refreshes the pool
// periodically.
package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

const schema = `
CREATE TABLE IF NOT EXISTS species_cache (
	id           INTEGER PRIMARY KEY,
	iconic_taxon TEXT NOT NULL DEFAULT '',
	genus_id     INTEGER NOT NULL DEFAULT 0,
	data         TEXT NOT NULL,
	fetched_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_species_cache_taxon ON species_cache(iconic_taxon);
CREATE INDEX IF NOT EXISTS idx_species_cache_genus ON species_cache(genus_id);
`

// Default cache tuning values.
const (
	// DefaultTTL is how long a cached species stays usable.
	DefaultTTL = 7 * 24 * time.Hour
	// DefaultWarmTarget is how many species per iconic taxon the warmer
	// keeps in the pool.
	DefaultWarmTarget = 60
	// DefaultWarmInterval is how often the background warmer runs.
	DefaultWarmInterval = 12 * time.Hour
)

// WarmTaxa lists the iconic taxa pre-loaded by the warmer; they mirror the
// categories offered in the frontend.
var WarmTaxa = []string{
	"Mammalia", "Aves", "Reptilia", "Amphibia",
	"Insecta", "Plantae", "Fungi",
}

// SpeciesCache is a caching decorator for ports.SpeciesRepository.
type SpeciesCache struct {
	db      *sql.DB
	source  ports.SpeciesRepository
	ttl     time.Duration
	placeID int
}

// Option configures the cache.
type Option func(*SpeciesCache)

// WithTTL sets how long cached entries stay fresh.
func WithTTL(ttl time.Duration) Option {
	return func(c *SpeciesCache) {
		c.ttl = ttl
	}
}

// WithPlaceID restricts warm-up queries to a geographic place.
func WithPlaceID(id int) Option {
	return func(c *SpeciesCache) {
		c.placeID = id
	}
}

// New creates a species cache on top of source, storing entries in db.
func New(db *sql.DB, source ports.SpeciesRepository, opts ...Option) (*SpeciesCache, error) {
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("applying cache schema: %w", err)
	}

	c := &SpeciesCache{
		db:     db,
		source: source,
		ttl:    DefaultTTL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// GetByID retrieves a species from the cache, falling back to the source.
func (c *SpeciesCache) GetByID(ctx context.Context, id int) (*species.Species, error) {
	var data string
	err := c.db.QueryRowContext(ctx,
		`SELECT data FROM species_cache WHERE id = ? AND fetched_at > ?`,
		id, c.freshnessCutoff()).Scan(&data)
	if err == nil {
		return decodeSpecies(data)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("querying species cache: %w", err)
	}

	sp, err := c.source.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c.store(ctx, sp)
	return sp, nil
}

// GetRandom serves random species from the cache; when the cache cannot
// satisfy the request it falls back to the source and stores the results.
func (c *SpeciesCache) GetRandom(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	cached, err := c.randomFromCache(ctx, filter)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 1
	}
	if len(cached) >= limit {
		return cached, nil
	}

	fromSource, err := c.source.GetRandom(ctx, filter)
	if err != nil {
		// The cache may still partially satisfy the request.
		if len(cached) > 0 {
			return cached, nil
		}
		return nil, err
	}
	for _, sp := range fromSource {
		c.store(ctx, sp)
	}
	return fromSource, nil
}

// GetSimilar serves same-genus species from the cache. For a species known
// to the cache the cached result is returned even when partial or empty:
// falling back to the source would cost two rate-limited API calls per
// question, and the question factory already tops up missing choices with
// same-taxon species. The source is only queried for species the cache has
// never seen.
func (c *SpeciesCache) GetSimilar(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
	var genusID int
	err := c.db.QueryRowContext(ctx,
		`SELECT genus_id FROM species_cache WHERE id = ?`, speciesID).Scan(&genusID)
	if err == nil {
		if genusID == 0 {
			return []*species.Species{}, nil
		}
		return c.similarFromCache(ctx, speciesID, genusID, limit)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("querying species cache: %w", err)
	}

	fromSource, err := c.source.GetSimilar(ctx, speciesID, limit)
	if err != nil {
		return nil, err
	}
	for _, sp := range fromSource {
		c.store(ctx, sp)
	}
	return fromSource, nil
}

// Search always queries the source: free-text search does not map well to
// the cached pool.
func (c *SpeciesCache) Search(ctx context.Context, query string, limit int) ([]*species.Species, error) {
	return c.source.Search(ctx, query, limit)
}

// Warm tops up the cache pool for each taxon up to perTaxon fresh entries.
func (c *SpeciesCache) Warm(ctx context.Context, taxa []string, perTaxon int) error {
	for _, taxon := range taxa {
		count, err := c.freshCount(ctx, taxon)
		if err != nil {
			return err
		}
		missing := perTaxon - count
		if missing <= 0 {
			continue
		}

		found, err := c.source.GetRandom(ctx, ports.SpeciesFilter{
			IconicTaxon: taxon,
			PlaceID:     c.placeID,
			Limit:       missing,
			HasPhotos:   true,
		})
		if err != nil {
			return fmt.Errorf("warming %s: %w", taxon, err)
		}
		for _, sp := range found {
			c.store(ctx, sp)
		}
	}
	return nil
}

// StartAutoWarm warms the cache immediately, then re-warms at every
// interval until ctx is cancelled. Errors are logged, not fatal: the cache
// degrades gracefully to source fallthrough.
func (c *SpeciesCache) StartAutoWarm(ctx context.Context, interval time.Duration, taxa []string, perTaxon int) {
	warm := func() {
		if err := c.Warm(ctx, taxa, perTaxon); err != nil {
			log.Printf("species cache warm-up: %v", err)
			return
		}
		log.Printf("species cache warmed (%d taxa, target %d each)", len(taxa), perTaxon)
	}

	warm()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			warm()
		}
	}
}

// randomFromCache picks fresh random entries matching the filter.
func (c *SpeciesCache) randomFromCache(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 1
	}

	query := `SELECT data FROM species_cache WHERE fetched_at > ?`
	args := []interface{}{c.freshnessCutoff()}

	if filter.IconicTaxon != "" {
		query += ` AND iconic_taxon = ?`
		args = append(args, filter.IconicTaxon)
	}
	if len(filter.ExcludeIDs) > 0 {
		query += ` AND id NOT IN (` + placeholders(len(filter.ExcludeIDs)) + `)`
		for _, id := range filter.ExcludeIDs {
			args = append(args, id)
		}
	}
	query += ` ORDER BY RANDOM() LIMIT ?`
	args = append(args, limit)

	return c.querySpecies(ctx, query, args...)
}

// similarFromCache picks fresh same-genus entries.
func (c *SpeciesCache) similarFromCache(ctx context.Context, speciesID, genusID, limit int) ([]*species.Species, error) {
	return c.querySpecies(ctx, `
		SELECT data FROM species_cache
		WHERE genus_id = ? AND id != ? AND fetched_at > ?
		ORDER BY RANDOM() LIMIT ?`,
		genusID, speciesID, c.freshnessCutoff(), limit)
}

// querySpecies runs a query returning data columns and decodes each row.
func (c *SpeciesCache) querySpecies(ctx context.Context, query string, args ...interface{}) ([]*species.Species, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying species cache: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]*species.Species, 0)
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scanning cached species: %w", err)
		}
		sp, err := decodeSpecies(data)
		if err != nil {
			return nil, err
		}
		result = append(result, sp)
	}
	return result, rows.Err()
}

// store upserts a species into the cache. Failures are logged, not
// returned: caching is best-effort and must never break a request.
func (c *SpeciesCache) store(ctx context.Context, sp *species.Species) {
	snap := sp.Snapshot()
	data, err := json.Marshal(snap)
	if err != nil {
		log.Printf("species cache: encoding species %d: %v", sp.ID(), err)
		return
	}

	genusID := sp.GenusID()

	_, err = c.db.ExecContext(ctx, `
		INSERT INTO species_cache (id, iconic_taxon, genus_id, data, fetched_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			iconic_taxon = excluded.iconic_taxon,
			genus_id = excluded.genus_id,
			data = excluded.data,
			fetched_at = excluded.fetched_at`,
		sp.ID(), sp.IconicTaxon(), genusID, string(data),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		log.Printf("species cache: storing species %d: %v", sp.ID(), err)
	}
}

// freshCount counts fresh entries for a taxon.
func (c *SpeciesCache) freshCount(ctx context.Context, taxon string) (int, error) {
	var count int
	err := c.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM species_cache
		WHERE iconic_taxon = ? AND fetched_at > ?`,
		taxon, c.freshnessCutoff()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting cached species: %w", err)
	}
	return count, nil
}

// freshnessCutoff returns the timestamp below which entries are stale.
func (c *SpeciesCache) freshnessCutoff() string {
	return time.Now().UTC().Add(-c.ttl).Format(time.RFC3339Nano)
}

// decodeSpecies rebuilds a domain species from its cached snapshot.
func decodeSpecies(data string) (*species.Species, error) {
	var snap species.Snapshot
	if err := json.Unmarshal([]byte(data), &snap); err != nil {
		return nil, fmt.Errorf("decoding cached species: %w", err)
	}
	return species.FromSnapshot(snap)
}

// placeholders returns n comma-separated SQL placeholders.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// Ensure interface compliance.
var _ ports.SpeciesRepository = (*SpeciesCache)(nil)

// String describes the cache configuration, useful for startup logs.
func (c *SpeciesCache) String() string {
	return "species cache (ttl " + strconv.Itoa(int(c.ttl.Hours())) + "h)"
}
