// Package mock provides mock implementations for development and testing.
package mock

import (
	"context"
	"math/rand"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// MockSpecies contains predefined species data for development.
var MockSpecies = []struct {
	ID          int
	Name        string
	CommonName  string
	IconicTaxon string
	PhotoURL    string
}{
	{1, "Vulpes vulpes", "Renard roux", "Mammalia", "https://static.inaturalist.org/photos/265916780/medium.jpg"},
	{2, "Canis lupus", "Loup gris", "Mammalia", "https://inaturalist-open-data.s3.amazonaws.com/photos/43410580/medium.jpg"},
	{3, "Cervus elaphus", "Cerf elaphe", "Mammalia", "https://inaturalist-open-data.s3.amazonaws.com/photos/25186445/medium.jpg"},
	{4, "Sus scrofa", "Sanglier", "Mammalia", "https://inaturalist-open-data.s3.amazonaws.com/photos/414500081/medium.jpg"},
	{5, "Meles meles", "Blaireau europeen", "Mammalia", "https://inaturalist-open-data.s3.amazonaws.com/photos/172054371/medium.jpg"},
	{6, "Pica pica", "Pie bavarde", "Aves", "https://inaturalist-open-data.s3.amazonaws.com/photos/103446096/medium.jpg"},
	{7, "Corvus corax", "Grand corbeau", "Aves", "https://inaturalist-open-data.s3.amazonaws.com/photos/54389687/medium.jpg"},
	{8, "Falco tinnunculus", "Faucon crecerelle", "Aves", "https://inaturalist-open-data.s3.amazonaws.com/photos/598044895/medium.jpg"},
	{9, "Buteo buteo", "Buse variable", "Aves", "https://inaturalist-open-data.s3.amazonaws.com/photos/192398523/medium.jpeg"},
	{10, "Erithacus rubecula", "Rouge-gorge familier", "Aves", "https://static.inaturalist.org/photos/331946615/medium.jpeg"},
	{11, "Lacerta viridis", "Lezard vert", "Reptilia", "https://inaturalist-open-data.s3.amazonaws.com/photos/74355963/medium.jpg"},
	{12, "Natrix natrix", "Couleuvre a collier", "Reptilia", "https://inaturalist-open-data.s3.amazonaws.com/photos/454734933/medium.jpg"},
	{13, "Rana temporaria", "Grenouille rousse", "Amphibia", "https://inaturalist-open-data.s3.amazonaws.com/photos/171173/medium.jpg"},
	{14, "Bufo bufo", "Crapaud commun", "Amphibia", "https://static.inaturalist.org/photos/585412909/medium.jpg"},
	{15, "Papilio machaon", "Machaon", "Insecta", "https://inaturalist-open-data.s3.amazonaws.com/photos/84864834/medium.jpg"},
	{16, "Aglais io", "Paon du jour", "Insecta", "https://inaturalist-open-data.s3.amazonaws.com/photos/4710619/medium.jpg"},
	{17, "Quercus robur", "Chene pedoncule", "Plantae", "https://inaturalist-open-data.s3.amazonaws.com/photos/482832455/medium.jpg"},
	{18, "Fagus sylvatica", "Hetre commun", "Plantae", "https://inaturalist-open-data.s3.amazonaws.com/photos/608828197/medium.jpg"},
	{19, "Amanita muscaria", "Amanite tue-mouches", "Fungi", "https://inaturalist-open-data.s3.amazonaws.com/photos/111494893/medium.jpg"},
	{20, "Boletus edulis", "Cepe de Bordeaux", "Fungi", "https://inaturalist-open-data.s3.amazonaws.com/photos/328303120/medium.jpg"},
}

// SpeciesRepository is a mock implementation of SpeciesRepository for development.
type SpeciesRepository struct {
	usedIDs map[int]bool
}

// NewSpeciesRepository creates a new mock species repository.
func NewSpeciesRepository() *SpeciesRepository {
	return &SpeciesRepository{
		usedIDs: make(map[int]bool),
	}
}

// ResetUsed clears the used species tracking.
func (r *SpeciesRepository) ResetUsed() {
	r.usedIDs = make(map[int]bool)
}

// GetByID returns a species by its ID.
func (r *SpeciesRepository) GetByID(_ context.Context, id int) (*species.Species, error) {
	for _, m := range MockSpecies {
		if m.ID == id {
			return r.createSpecies(m.ID, m.Name, m.CommonName, m.IconicTaxon, m.PhotoURL), nil
		}
	}
	// Return first species if not found
	m := MockSpecies[0]
	return r.createSpecies(m.ID, m.Name, m.CommonName, m.IconicTaxon, m.PhotoURL), nil
}

// GetRandom returns random species, avoiding recently used ones.
func (r *SpeciesRepository) GetRandom(_ context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	available := r.getAvailableSpecies(filter)
	if len(available) == 0 {
		// Reset if all species used
		r.usedIDs = make(map[int]bool)
		available = r.getAvailableSpecies(filter)
	}

	// Shuffle available species
	rand.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	limit := filter.Limit
	if limit <= 0 || limit > len(available) {
		limit = len(available)
	}

	result := make([]*species.Species, 0, limit)
	for i := 0; i < limit && i < len(available); i++ {
		m := available[i]
		sp := r.createSpecies(m.ID, m.Name, m.CommonName, m.IconicTaxon, m.PhotoURL)
		r.usedIDs[m.ID] = true
		result = append(result, sp)
	}

	return result, nil
}

// GetSimilar returns species in the same taxon.
func (r *SpeciesRepository) GetSimilar(_ context.Context, speciesID int, limit int) ([]*species.Species, error) {
	var targetTaxon string
	for _, m := range MockSpecies {
		if m.ID == speciesID {
			targetTaxon = m.IconicTaxon
			break
		}
	}

	result := make([]*species.Species, 0, limit)
	for _, m := range MockSpecies {
		if m.IconicTaxon == targetTaxon && m.ID != speciesID {
			sp := r.createSpecies(m.ID, m.Name, m.CommonName, m.IconicTaxon, m.PhotoURL)
			result = append(result, sp)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// Search searches for species by name.
func (r *SpeciesRepository) Search(_ context.Context, query string, limit int) ([]*species.Species, error) {
	result := make([]*species.Species, 0, limit)
	for _, m := range MockSpecies {
		sp := r.createSpecies(m.ID, m.Name, m.CommonName, m.IconicTaxon, m.PhotoURL)
		result = append(result, sp)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (r *SpeciesRepository) getAvailableSpecies(filter ports.SpeciesFilter) []struct {
	ID          int
	Name        string
	CommonName  string
	IconicTaxon string
	PhotoURL    string
} {
	available := make([]struct {
		ID          int
		Name        string
		CommonName  string
		IconicTaxon string
		PhotoURL    string
	}, 0)

	for _, m := range MockSpecies {
		// Skip used species
		if r.usedIDs[m.ID] {
			continue
		}
		// Filter by taxon if specified
		if filter.IconicTaxon != "" && m.IconicTaxon != filter.IconicTaxon {
			continue
		}
		// Filter excluded IDs
		excluded := false
		for _, id := range filter.ExcludeIDs {
			if m.ID == id {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		available = append(available, m)
	}

	return available
}

func (r *SpeciesRepository) createSpecies(id int, name, commonName, iconicTaxon, photoURL string) *species.Species {
	sp, _ := species.New(id, name, commonName, iconicTaxon)
	sp.AddPhoto(species.Photo{
		ID:        id * 100,
		URL:       photoURL,
		MediumURL: photoURL,
		LargeURL:  photoURL,
	})
	return sp
}

// Ensure interface compliance
var _ ports.SpeciesRepository = (*SpeciesRepository)(nil)
