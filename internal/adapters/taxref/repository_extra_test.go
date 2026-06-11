package taxref_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

func seedTaxref(t *testing.T) (*sql.DB, *taxref.Repository) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	// A few birds of distinct families + a mammal.
	_, err = db.Exec(`INSERT INTO taxref_species
		(cd_nom, cd_ref, rang, scientific_name, vernacular_name, kingdom, class, ordre, family, genus, taxa_group, fr) VALUES
		(1, 1, 'ES', 'Erithacus rubecula', 'Rougegorge', 'Animalia', 'Aves', 'Passeriformes', 'Muscicapidae', 'Erithacus', 'Oiseaux', 'P'),
		(2, 2, 'ES', 'Turdus merula', 'Merle noir', 'Animalia', 'Aves', 'Passeriformes', 'Turdidae', 'Turdus', 'Oiseaux', 'P'),
		(3, 3, 'ES', 'Parus major', 'Mesange', 'Animalia', 'Aves', 'Passeriformes', 'Paridae', 'Parus', 'Oiseaux', 'P'),
		(4, 4, 'ES', 'Vulpes vulpes', 'Renard', 'Animalia', 'Mammalia', 'Carnivora', 'Canidae', 'Vulpes', 'Mammiferes', 'P')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db, taxref.NewRepository(db)
}

func TestTaxrefRepository_QueriesAndMedia(t *testing.T) {
	db, r := seedTaxref(t)
	ctx := context.Background()

	// Lookups.
	if sp, err := r.GetByID(ctx, 1); err != nil || sp.ScientificName() != "Erithacus rubecula" || sp.Family() != "Muscicapidae" {
		t.Fatalf("GetByID = %v, %v", sp, err)
	}
	if res, err := r.Search(ctx, "merle", 5); err != nil || len(res) == 0 {
		t.Errorf("Search = %d, %v", len(res), err)
	}
	if sim, err := r.GetSimilar(ctx, 1, 5); err != nil {
		t.Errorf("GetSimilar: %v", err)
	} else if len(sim) == 0 {
		t.Log("no same-family neighbour (ok with this tiny set)")
	}
	// Other families (for the family quiz): birds from different families.
	if other, err := r.GetOtherFamilies(ctx, 1, 3); err != nil || len(other) == 0 {
		t.Errorf("GetOtherFamilies = %d, %v", len(other), err)
	}

	// Photos.
	id, err := r.AddPhoto(ctx, 1, "https://x/p.jpg", "auteur", "cc-by", "beginner")
	if err != nil {
		t.Fatalf("AddPhoto: %v", err)
	}
	if list, _ := r.ListPhotos(ctx, 1); len(list) != 1 {
		t.Errorf("ListPhotos = %d", len(list))
	}
	if n, _ := r.CountPhotos(ctx); n != 1 {
		t.Errorf("CountPhotos = %d", n)
	}
	if n, _ := r.CountSpeciesWithPhotos(ctx); n != 1 {
		t.Errorf("CountSpeciesWithPhotos = %d", n)
	}
	if cov, _ := r.PhotoCoverage(ctx); len(cov) == 0 {
		t.Error("PhotoCoverage empty")
	}
	// A random species with photos must be the one we illustrated.
	if got, err := r.GetRandom(ctx, ports.SpeciesFilter{Limit: 1, HasPhotos: true}); err != nil || len(got) == 0 || got[0].ID() != 1 {
		t.Errorf("GetRandom photo = %v, %v", got, err)
	}

	// Zones on the photo.
	if err := r.SetPhotoZones(ctx, id, `{"zoom":{"x":0.1,"y":0.1,"w":0.4,"h":0.4},"species":[{"cd_nom":1,"name":"R","x":0.1,"y":0.1,"w":0.3,"h":0.3}]}`); err != nil {
		t.Fatalf("SetPhotoZones: %v", err)
	}
	if zp, _ := r.PhotosWithSpeciesZones(ctx, 10); len(zp) != 1 {
		t.Errorf("PhotosWithSpeciesZones = %d", len(zp))
	}
	if zones, _ := r.PhotoSpeciesZones(ctx, id); len(zones) != 1 {
		t.Errorf("PhotoSpeciesZones = %d", len(zones))
	}
	if u, err := r.PhotoURLByID(ctx, id); err != nil || u == "" {
		t.Errorf("PhotoURLByID = %q, %v", u, err)
	}

	// Sounds.
	sid, err := r.AddSound(ctx, 2, "https://x/s.mp3", "rec", "cc-by")
	if err != nil {
		t.Fatalf("AddSound: %v", err)
	}
	if ls, _ := r.ListSounds(ctx, 2); len(ls) != 1 {
		t.Errorf("ListSounds = %d", len(ls))
	}
	if cd, _, _, _, ok, _ := r.RandomSounded(ctx); !ok || cd != 2 {
		t.Errorf("RandomSounded = %d,%v", cd, ok)
	}
	if _, err := r.DeleteSound(ctx, sid); err != nil {
		t.Errorf("DeleteSound: %v", err)
	}

	// Delete the photo (returns its URL).
	if u, err := r.DeletePhoto(ctx, id); err != nil || u == "" {
		t.Errorf("DeletePhoto = %q, %v", u, err)
	}

	// Occurrence-based filters once data is present.
	_, _ = db.Exec(`INSERT INTO species_months (cd_nom, month) VALUES (1, 6)`)
	_, _ = db.Exec(`INSERT INTO species_regions (cd_nom, region) VALUES (1, 'Bretagne')`)
	if regions, _ := r.ListRegions(ctx); len(regions) != 1 || regions[0] != "Bretagne" {
		t.Errorf("ListRegions = %v", regions)
	}
	if n, _ := r.CountSpecies(ctx); n != 4 {
		t.Errorf("CountSpecies = %d", n)
	}
}

func TestTaxref_MetaAndPhotoCSV(t *testing.T) {
	db, r := seedTaxref(t)
	ctx := context.Background()
	if err := taxref.SetMeta(db, "version", "v18.0"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if v := r.Version(ctx); v != "v18.0" {
		t.Errorf("Version = %q", v)
	}
	// ParsePhotoCSV on a missing file errors cleanly.
	if _, err := taxref.ParsePhotoCSV("/no/such/file.csv"); err == nil {
		t.Error("ParsePhotoCSV missing file should error")
	}
}
