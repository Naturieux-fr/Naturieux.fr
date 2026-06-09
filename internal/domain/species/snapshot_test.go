package species_test

import (
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

func TestSpecies_SnapshotRoundTrip(t *testing.T) {
	sp, err := species.New(42, "Vulpes vulpes", "Renard roux", "Mammalia")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	sp.SetRank("species")
	sp.SetAncestorIDs([]int{1, 2, 3})
	sp.AddPhoto(species.Photo{
		ID:          7,
		MediumURL:   "https://example.com/m.jpg",
		Attribution: "(c) Someone, CC BY",
		LicenseCode: "cc-by",
	})

	restored, err := species.FromSnapshot(sp.Snapshot())
	if err != nil {
		t.Fatalf("FromSnapshot() error = %v", err)
	}

	if restored.ID() != 42 {
		t.Errorf("ID = %d, want 42", restored.ID())
	}
	if restored.ScientificName() != "Vulpes vulpes" {
		t.Errorf("ScientificName = %s, want Vulpes vulpes", restored.ScientificName())
	}
	if restored.CommonName() != "Renard roux" {
		t.Errorf("CommonName = %s, want Renard roux", restored.CommonName())
	}
	if restored.Rank() != "species" {
		t.Errorf("Rank = %s, want species", restored.Rank())
	}
	if len(restored.AncestorIDs()) != 3 {
		t.Errorf("AncestorIDs = %v, want 3 elements", restored.AncestorIDs())
	}
	photos := restored.Photos()
	if len(photos) != 1 {
		t.Fatalf("Photos = %d, want 1", len(photos))
	}
	if photos[0].LicenseCode != "cc-by" || photos[0].Attribution == "" {
		t.Errorf("Photo credit lost: %+v", photos[0])
	}
}

func TestFromSnapshot_Invalid(t *testing.T) {
	_, err := species.FromSnapshot(species.Snapshot{ID: 0, ScientificName: ""})
	if err == nil {
		t.Error("FromSnapshot() with invalid data should return an error")
	}
}
