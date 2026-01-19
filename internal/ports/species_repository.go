// Package ports defines the interfaces (ports) for the application.
package ports

import (
	"context"

	"github.com/fieve/naturieux/internal/domain/species"
)

// SpeciesFilter defines filtering options for species queries.
type SpeciesFilter struct {
	IconicTaxon string // Filter by iconic taxon (e.g., "Mammalia")
	PlaceID     int    // Filter by geographic location
	Limit       int    // Maximum number of results
	HasPhotos   bool   // Only species with photos
	Quality     string // Quality grade (research, needs_id, casual)
	ExcludeIDs  []int  // Species IDs to exclude
}

// SpeciesRepository defines the interface for species data access.
type SpeciesRepository interface {
	// GetByID retrieves a species by its ID.
	GetByID(ctx context.Context, id int) (*species.Species, error)

	// GetRandom retrieves random species matching the filter.
	GetRandom(ctx context.Context, filter SpeciesFilter) ([]*species.Species, error)

	// GetSimilar retrieves species similar to the given one (same family/genus).
	GetSimilar(ctx context.Context, speciesID int, limit int) ([]*species.Species, error)

	// Search searches for species by name.
	Search(ctx context.Context, query string, limit int) ([]*species.Species, error)
}
