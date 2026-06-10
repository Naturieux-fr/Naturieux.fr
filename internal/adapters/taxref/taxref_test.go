package taxref_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// sampleTAXREF is a minimal Darwin Core extract: a header plus a few rows
// covering a valid species, a synonym, a non-species rank, and two families.
const sampleTAXREF = "taxonID\tacceptedNameUsageID\tparentNameUsageID\tscientificName\tkingdom\tclass\torder\tfamily\tgenus\ttaxonRank\tvernacularName\n" +
	// Valid mammals (Canidae / Vulpes)
	"60585\t60585\t198937\tVulpes vulpes\tAnimalia\tMammalia\tCarnivora\tCanidae\tVulpes\tspecies\tRenard roux, Renard, Goupil\n" +
	"60577\t60577\t198937\tCanis lupus\tAnimalia\tMammalia\tCarnivora\tCanidae\tCanis\tspecies\tLoup gris\n" +
	"60596\t60596\t198937\tMeles meles\tAnimalia\tMammalia\tCarnivora\tMustelidae\tMeles\tspecies\tBlaireau européen\n" +
	// A bird
	"3371\t3371\t2222\tColumba palumbus\tAnimalia\tAves\tColumbiformes\tColumbidae\tColumba\tspecies\tPigeon ramier\n" +
	// A synonym (taxonID != acceptedNameUsageID) — must be skipped
	"99999\t60585\t198937\tVulpes vulgaris\tAnimalia\tMammalia\tCarnivora\tCanidae\tVulpes\tspecies\tvieux synonyme\n" +
	// A genus rank — must be skipped (not a species)
	"198937\t198937\t0\tVulpes\tAnimalia\tMammalia\tCarnivora\tCanidae\tVulpes\tgenus\t\n"

func newTestRepo(t *testing.T) *taxref.Repository {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "taxref.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	stats, err := taxref.Import(db, strings.NewReader(sampleTAXREF))
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	// 6 data rows: 4 valid species + 1 synonym + 1 genus
	if stats.Imported != 4 {
		t.Fatalf("Import() imported %d, want 4 valid species (skipped %d)", stats.Imported, stats.Skipped)
	}
	return taxref.NewRepository(db)
}

func TestImport_FiltersSynonymsAndRanks(t *testing.T) {
	repo := newTestRepo(t)

	// The synonym (99999) and the genus (198937) must not be queryable
	if _, err := repo.GetByID(context.Background(), 99999); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("synonym should be skipped, got err = %v", err)
	}
	if _, err := repo.GetByID(context.Background(), 198937); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("genus rank should be skipped, got err = %v", err)
	}
}

func TestImport_MissingColumn(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "taxref.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}

	bad := "taxonID\tscientificName\n60585\tVulpes vulpes\n" // no acceptedNameUsageID / taxonRank
	if _, err := taxref.Import(db, strings.NewReader(bad)); err == nil {
		t.Error("Import() should fail when required columns are missing")
	}
}

func TestRepository_GetByID(t *testing.T) {
	repo := newTestRepo(t)

	sp, err := repo.GetByID(context.Background(), 60585)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if sp.ScientificName() != "Vulpes vulpes" {
		t.Errorf("ScientificName = %s, want Vulpes vulpes", sp.ScientificName())
	}
	// Vernacular keeps only the first name
	if sp.CommonName() != "Renard roux" {
		t.Errorf("CommonName = %s, want Renard roux", sp.CommonName())
	}
	if sp.IconicTaxon() != "Mammalia" {
		t.Errorf("IconicTaxon = %s, want Mammalia", sp.IconicTaxon())
	}
}

func TestRepository_GetRandom_FilterByCategory(t *testing.T) {
	repo := newTestRepo(t)

	birds, err := repo.GetRandom(context.Background(), ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 5})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(birds) != 1 {
		t.Fatalf("GetRandom(Aves) = %d species, want 1", len(birds))
	}
	if birds[0].ScientificName() != "Columba palumbus" {
		t.Errorf("got %s, want Columba palumbus", birds[0].ScientificName())
	}

	mammals, err := repo.GetRandom(context.Background(), ports.SpeciesFilter{IconicTaxon: "Mammalia", Limit: 10})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(mammals) != 3 {
		t.Errorf("GetRandom(Mammalia) = %d species, want 3", len(mammals))
	}
}

func TestRepository_GetRandom_HasPhotosFiltersToOwned(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// No photos yet: requiring photos returns nothing
	got, err := repo.GetRandom(ctx, ports.SpeciesFilter{HasPhotos: true, Limit: 10})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetRandom(HasPhotos) = %d, want 0 before any photo", len(got))
	}

	// Add a photo for the fox; now it is the only photo-bearing species
	if err := repo.AddPhoto(ctx, 60585, "https://nat.example/fox.jpg", "(c) Moi", "cc-by"); err != nil {
		t.Fatalf("AddPhoto() error = %v", err)
	}
	got, err = repo.GetRandom(ctx, ports.SpeciesFilter{HasPhotos: true, Limit: 10})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(got) != 1 || got[0].ID() != 60585 {
		t.Fatalf("GetRandom(HasPhotos) = %d species, want only the fox", len(got))
	}
	if !got[0].HasPhotos() {
		t.Error("returned species should carry its photo")
	}
	if got[0].Photos()[0].LicenseCode != "cc-by" {
		t.Errorf("photo license = %s, want cc-by", got[0].Photos()[0].LicenseCode)
	}
}

func TestRepository_GetRandom_ExcludeIDs(t *testing.T) {
	repo := newTestRepo(t)

	got, err := repo.GetRandom(context.Background(), ports.SpeciesFilter{
		IconicTaxon: "Mammalia", Limit: 10, ExcludeIDs: []int{60585, 60577},
	})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}
	if len(got) != 1 || got[0].ID() != 60596 {
		t.Errorf("GetRandom(exclude) = %v, want only Meles meles (60596)", got)
	}
}

func TestRepository_GetSimilar_PrefersGenusThenFamily(t *testing.T) {
	repo := newTestRepo(t)

	// Canis lupus (60577): same family Canidae has Vulpes vulpes; no other
	// Canis. So the family fallback should surface the fox.
	similar, err := repo.GetSimilar(context.Background(), 60577, 3)
	if err != nil {
		t.Fatalf("GetSimilar() error = %v", err)
	}
	if len(similar) == 0 {
		t.Fatal("GetSimilar() returned no distractors")
	}
	for _, sp := range similar {
		if sp.ID() == 60577 {
			t.Error("GetSimilar() must not include the target species")
		}
		if sp.IconicTaxon() != "Mammalia" {
			t.Errorf("distractor %s is not a mammal", sp.ScientificName())
		}
	}
}

func TestRepository_Search(t *testing.T) {
	repo := newTestRepo(t)

	byScientific, err := repo.Search(context.Background(), "Vulpes", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(byScientific) != 1 || byScientific[0].ID() != 60585 {
		t.Errorf("Search(Vulpes) = %d results, want the fox", len(byScientific))
	}

	byVernacular, err := repo.Search(context.Background(), "Loup", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(byVernacular) != 1 || byVernacular[0].ScientificName() != "Canis lupus" {
		t.Errorf("Search(Loup) did not find Canis lupus")
	}
}
