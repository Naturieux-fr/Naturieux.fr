package cache_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/cache"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// stubSource is an instrumented fake species source.
type stubSource struct {
	calls      map[string]int
	random     []*species.Species
	similar    []*species.Species
	byID       map[int]*species.Species
	failRandom bool
}

func newStubSource() *stubSource {
	return &stubSource{
		calls: make(map[string]int),
		byID:  make(map[int]*species.Species),
	}
}

func (s *stubSource) GetByID(_ context.Context, id int) (*species.Species, error) {
	s.calls["GetByID"]++
	if sp, ok := s.byID[id]; ok {
		return sp, nil
	}
	return nil, ports.ErrNotFound
}

func (s *stubSource) GetRandom(_ context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	s.calls["GetRandom"]++
	if s.failRandom {
		return nil, errors.New("api unavailable")
	}
	limit := filter.Limit
	if limit <= 0 || limit > len(s.random) {
		limit = len(s.random)
	}
	return s.random[:limit], nil
}

func (s *stubSource) GetSimilar(_ context.Context, _ int, limit int) ([]*species.Species, error) {
	s.calls["GetSimilar"]++
	if limit > len(s.similar) {
		limit = len(s.similar)
	}
	return s.similar[:limit], nil
}

func (s *stubSource) Search(_ context.Context, _ string, _ int) ([]*species.Species, error) {
	s.calls["Search"]++
	return nil, nil
}

func makeSpecies(t *testing.T, id int, name, taxon string, ancestors ...int) *species.Species {
	t.Helper()
	sp, err := species.New(id, name, name+" common", taxon)
	if err != nil {
		t.Fatalf("species.New() error = %v", err)
	}
	sp.SetAncestorIDs(ancestors)
	sp.AddPhoto(species.Photo{
		ID:          id,
		MediumURL:   "https://example.com/photo.jpg",
		Attribution: "(c) Someone, CC BY",
		LicenseCode: "cc-by",
	})
	return sp
}

func openTestCache(t *testing.T, source ports.SpeciesRepository, opts ...cache.Option) (*cache.SpeciesCache, *sql.DB) {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	c, err := cache.New(db, source, opts...)
	if err != nil {
		t.Fatalf("cache.New() error = %v", err)
	}
	return c, db
}

func TestSpeciesCache_GetRandom_FallsThroughAndCaches(t *testing.T) {
	source := newStubSource()
	source.random = []*species.Species{
		makeSpecies(t, 1, "Vulpes vulpes", "Mammalia", 100),
		makeSpecies(t, 2, "Canis lupus", "Mammalia", 100),
	}
	c, _ := openTestCache(t, source)
	ctx := context.Background()

	// First call: cache empty, source is hit
	got, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Mammalia", Limit: 2})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetRandom() = %d species, want 2", len(got))
	}
	if source.calls["GetRandom"] != 1 {
		t.Errorf("source calls = %d, want 1", source.calls["GetRandom"])
	}

	// Second call: served entirely from cache, source not hit again
	got, err = c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Mammalia", Limit: 2})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetRandom() from cache = %d species, want 2", len(got))
	}
	if source.calls["GetRandom"] != 1 {
		t.Errorf("source calls after cache hit = %d, want still 1", source.calls["GetRandom"])
	}

	// Cached species keep their photo credit
	if got[0].Photos()[0].LicenseCode != "cc-by" {
		t.Error("cached species lost its photo license")
	}
}

func TestSpeciesCache_GetRandom_RespectsExcludeIDs(t *testing.T) {
	source := newStubSource()
	source.random = []*species.Species{
		makeSpecies(t, 1, "A", "Aves", 10),
		makeSpecies(t, 2, "B", "Aves", 10),
		makeSpecies(t, 3, "C", "Aves", 10),
	}
	c, _ := openTestCache(t, source)
	ctx := context.Background()

	if _, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 3}); err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	got, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 2, ExcludeIDs: []int{1}})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	for _, sp := range got {
		if sp.ID() == 1 {
			t.Error("GetRandom() returned an excluded species")
		}
	}
}

func TestSpeciesCache_GetRandom_SourceFailureFallsBackToPartialCache(t *testing.T) {
	source := newStubSource()
	source.random = []*species.Species{makeSpecies(t, 1, "A", "Aves", 10)}
	c, _ := openTestCache(t, source)
	ctx := context.Background()

	// Seed the cache with one species
	if _, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 1}); err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	// Ask for more than cached while the source is down: partial result
	source.failRandom = true
	got, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 5})
	if err != nil {
		t.Fatalf("GetRandom() should degrade gracefully, error = %v", err)
	}
	if len(got) != 1 {
		t.Errorf("GetRandom() = %d species, want 1 from cache", len(got))
	}
}

func TestSpeciesCache_GetByID_CachesAfterMiss(t *testing.T) {
	source := newStubSource()
	source.byID[7] = makeSpecies(t, 7, "Bubo bubo", "Aves", 50)
	c, _ := openTestCache(t, source)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		sp, err := c.GetByID(ctx, 7)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if sp.ScientificName() != "Bubo bubo" {
			t.Errorf("ScientificName = %s, want Bubo bubo", sp.ScientificName())
		}
	}
	if source.calls["GetByID"] != 1 {
		t.Errorf("source GetByID calls = %d, want 1", source.calls["GetByID"])
	}
}

func TestSpeciesCache_GetSimilar_ServedFromGenus(t *testing.T) {
	source := newStubSource()
	source.random = []*species.Species{
		makeSpecies(t, 1, "Falco tinnunculus", "Aves", 99),
		makeSpecies(t, 2, "Falco peregrinus", "Aves", 99),
		makeSpecies(t, 3, "Falco subbuteo", "Aves", 99),
	}
	c, _ := openTestCache(t, source)
	ctx := context.Background()

	// Seed the cache: all three share genus 99
	if _, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 3}); err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	similar, err := c.GetSimilar(ctx, 1, 2)
	if err != nil {
		t.Fatalf("GetSimilar() error = %v", err)
	}
	if len(similar) != 2 {
		t.Fatalf("GetSimilar() = %d species, want 2", len(similar))
	}
	for _, sp := range similar {
		if sp.ID() == 1 {
			t.Error("GetSimilar() must not return the target species")
		}
	}
	if source.calls["GetSimilar"] != 0 {
		t.Errorf("source GetSimilar calls = %d, want 0 (cache hit)", source.calls["GetSimilar"])
	}
}

func TestSpeciesCache_GetSimilar_FallsBackToSourceForUnknownSpecies(t *testing.T) {
	source := newStubSource()
	source.similar = []*species.Species{
		makeSpecies(t, 5, "Similar A", "Aves", 77),
		makeSpecies(t, 6, "Similar B", "Aves", 77),
	}
	c, _ := openTestCache(t, source)

	// Species 999 has never been cached: the source is queried
	similar, err := c.GetSimilar(context.Background(), 999, 2)
	if err != nil {
		t.Fatalf("GetSimilar() error = %v", err)
	}
	if len(similar) != 2 {
		t.Errorf("GetSimilar() = %d species, want 2 from source", len(similar))
	}
	if source.calls["GetSimilar"] != 1 {
		t.Errorf("source GetSimilar calls = %d, want 1", source.calls["GetSimilar"])
	}
}

func TestSpeciesCache_GetSimilar_KnownSpeciesNeverHitsSource(t *testing.T) {
	source := newStubSource()
	// A single cached species with no genus siblings
	source.random = []*species.Species{makeSpecies(t, 1, "Lonely", "Aves", 99)}
	c, _ := openTestCache(t, source)
	ctx := context.Background()

	if _, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 1}); err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	// Cached species without siblings: empty result, no API call — the
	// question factory tops up choices from the same taxon instead.
	similar, err := c.GetSimilar(ctx, 1, 3)
	if err != nil {
		t.Fatalf("GetSimilar() error = %v", err)
	}
	if len(similar) != 0 {
		t.Errorf("GetSimilar() = %d species, want 0", len(similar))
	}
	if source.calls["GetSimilar"] != 0 {
		t.Errorf("source GetSimilar calls = %d, want 0 (known species)", source.calls["GetSimilar"])
	}
}

func TestSpeciesCache_TTL_ExpiresEntries(t *testing.T) {
	source := newStubSource()
	source.random = []*species.Species{makeSpecies(t, 1, "A", "Aves", 10)}
	// Everything written is immediately stale
	c, _ := openTestCache(t, source, cache.WithTTL(-time.Second))
	ctx := context.Background()

	if _, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 1}); err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if _, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 1}); err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	// Both calls must have hit the source: stale entries are not served
	if source.calls["GetRandom"] != 2 {
		t.Errorf("source calls = %d, want 2 (stale cache ignored)", source.calls["GetRandom"])
	}
}

func TestSpeciesCache_Warm_FillsPoolPerTaxon(t *testing.T) {
	source := newStubSource()
	source.random = []*species.Species{
		makeSpecies(t, 1, "A", "Aves", 10),
		makeSpecies(t, 2, "B", "Aves", 10),
	}
	c, _ := openTestCache(t, source)
	ctx := context.Background()

	if err := c.Warm(ctx, []string{"Aves"}, 2); err != nil {
		t.Fatalf("Warm() error = %v", err)
	}
	if source.calls["GetRandom"] != 1 {
		t.Errorf("source calls = %d, want 1", source.calls["GetRandom"])
	}

	// Pool already full: warming again does not hit the source
	if err := c.Warm(ctx, []string{"Aves"}, 2); err != nil {
		t.Fatalf("Warm() error = %v", err)
	}
	if source.calls["GetRandom"] != 1 {
		t.Errorf("source calls after full pool = %d, want still 1", source.calls["GetRandom"])
	}

	// The pool serves quiz reads without the source
	got, err := c.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 2})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(got) != 2 || source.calls["GetRandom"] != 1 {
		t.Errorf("warmed pool should serve reads; got %d species, %d source calls",
			len(got), source.calls["GetRandom"])
	}
}
