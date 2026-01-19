package species_test

import (
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

func TestNewSpecies(t *testing.T) {
	tests := []struct {
		name        string
		id          int
		scientificN string
		commonName  string
		iconicTaxon string
		wantErr     bool
	}{
		{
			name:        "valid species",
			id:          42069,
			scientificN: "Vulpes vulpes",
			commonName:  "Renard roux",
			iconicTaxon: "Mammalia",
			wantErr:     false,
		},
		{
			name:        "missing scientific name",
			id:          1,
			scientificN: "",
			commonName:  "Test",
			iconicTaxon: "Mammalia",
			wantErr:     true,
		},
		{
			name:        "invalid id",
			id:          0,
			scientificN: "Test species",
			commonName:  "Test",
			iconicTaxon: "Mammalia",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := species.New(tt.id, tt.scientificN, tt.commonName, tt.iconicTaxon)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && s == nil {
				t.Error("New() returned nil species without error")
			}
		})
	}
}

func TestSpecies_AddPhoto(t *testing.T) {
	s, _ := species.New(1, "Test species", "Test", "Mammalia")

	photo := species.Photo{
		ID:        123,
		URL:       "https://example.com/photo.jpg",
		MediumURL: "https://example.com/photo_medium.jpg",
		LargeURL:  "https://example.com/photo_large.jpg",
	}

	s.AddPhoto(photo)

	if len(s.Photos()) != 1 {
		t.Errorf("AddPhoto() photos count = %d, want 1", len(s.Photos()))
	}

	if s.Photos()[0].ID != 123 {
		t.Errorf("AddPhoto() photo ID = %d, want 123", s.Photos()[0].ID)
	}
}

func TestSpecies_HasPhotos(t *testing.T) {
	s, _ := species.New(1, "Test species", "Test", "Mammalia")

	if s.HasPhotos() {
		t.Error("HasPhotos() = true, want false for new species")
	}

	s.AddPhoto(species.Photo{ID: 1, URL: "https://example.com/photo.jpg"})

	if !s.HasPhotos() {
		t.Error("HasPhotos() = false, want true after adding photo")
	}
}

func TestSpecies_DisplayName(t *testing.T) {
	s, _ := species.New(1, "Vulpes vulpes", "Renard roux", "Mammalia")

	if s.DisplayName() != "Renard roux" {
		t.Errorf("DisplayName() = %s, want Renard roux", s.DisplayName())
	}

	s2, _ := species.New(2, "Vulpes zerda", "", "Mammalia")
	if s2.DisplayName() != "Vulpes zerda" {
		t.Errorf("DisplayName() = %s, want Vulpes zerda (fallback to scientific)", s2.DisplayName())
	}
}

func TestIconicTaxon_IsValid(t *testing.T) {
	validTaxa := []string{
		"Mammalia", "Aves", "Reptilia", "Amphibia",
		"Actinopterygii", "Insecta", "Arachnida",
		"Mollusca", "Plantae", "Fungi",
	}

	for _, taxon := range validTaxa {
		if !species.IsValidIconicTaxon(taxon) {
			t.Errorf("IsValidIconicTaxon(%s) = false, want true", taxon)
		}
	}

	if species.IsValidIconicTaxon("Invalid") {
		t.Error("IsValidIconicTaxon(Invalid) = true, want false")
	}
}

func TestSpecies_Getters(t *testing.T) {
	s, _ := species.New(42, "Vulpes vulpes", "Red Fox", "Mammalia")

	if s.ID() != 42 {
		t.Errorf("ID() = %d, want 42", s.ID())
	}

	if s.ScientificName() != "Vulpes vulpes" {
		t.Errorf("ScientificName() = %s, want Vulpes vulpes", s.ScientificName())
	}

	if s.CommonName() != "Red Fox" {
		t.Errorf("CommonName() = %s, want Red Fox", s.CommonName())
	}

	if s.IconicTaxon() != "Mammalia" {
		t.Errorf("IconicTaxon() = %s, want Mammalia", s.IconicTaxon())
	}
}

func TestSpecies_AncestorIDs(t *testing.T) {
	s, _ := species.New(1, "Test", "Test", "Mammalia")

	// Initially empty
	if len(s.AncestorIDs()) != 0 {
		t.Error("AncestorIDs() should be empty initially")
	}

	// Set ancestors
	ancestors := []int{1, 2, 3, 4, 5}
	s.SetAncestorIDs(ancestors)

	if len(s.AncestorIDs()) != 5 {
		t.Errorf("AncestorIDs() length = %d, want 5", len(s.AncestorIDs()))
	}

	if s.AncestorIDs()[0] != 1 {
		t.Errorf("AncestorIDs()[0] = %d, want 1", s.AncestorIDs()[0])
	}
}

func TestSpecies_Rank(t *testing.T) {
	s, _ := species.New(1, "Test", "Test", "Mammalia")

	// Initially empty
	if s.Rank() != "" {
		t.Errorf("Rank() = %s, want empty string initially", s.Rank())
	}

	// Set rank
	s.SetRank("species")

	if s.Rank() != "species" {
		t.Errorf("Rank() = %s, want species", s.Rank())
	}
}

func TestSpecies_NegativeID(t *testing.T) {
	_, err := species.New(-1, "Test", "Test", "Mammalia")
	if err == nil {
		t.Error("New() should return error for negative ID")
	}
}

func TestSpecies_MultiplePhotos(t *testing.T) {
	s, _ := species.New(1, "Test", "Test", "Mammalia")

	s.AddPhoto(species.Photo{ID: 1, URL: "url1"})
	s.AddPhoto(species.Photo{ID: 2, URL: "url2"})
	s.AddPhoto(species.Photo{ID: 3, URL: "url3"})

	if len(s.Photos()) != 3 {
		t.Errorf("Photos() count = %d, want 3", len(s.Photos()))
	}
}
