// Package species contains domain entities for species and taxa.
package species

import (
	"errors"
)

// IconicTaxon represents the major taxonomic groups.
var validIconicTaxa = map[string]bool{
	"Mammalia":       true,
	"Aves":           true,
	"Reptilia":       true,
	"Amphibia":       true,
	"Actinopterygii": true,
	"Insecta":        true,
	"Arachnida":      true,
	"Mollusca":       true,
	"Plantae":        true,
	"Fungi":          true,
}

// IsValidIconicTaxon checks if a taxon name is valid.
func IsValidIconicTaxon(taxon string) bool {
	return validIconicTaxa[taxon]
}

// Photo represents a species photo from iNaturalist.
type Photo struct {
	ID          int
	URL         string
	MediumURL   string
	LargeURL    string
	OriginalURL string
	SquareURL   string
	Attribution string
	LicenseCode string
}

// Species represents a biological species entity.
type Species struct {
	id             int
	scientificName string
	commonName     string
	iconicTaxon    string
	photos         []Photo
	ancestorIDs    []int
	rank           string
}

// New creates a new Species with validation.
func New(id int, scientificName, commonName, iconicTaxon string) (*Species, error) {
	if id <= 0 {
		return nil, errors.New("species id must be positive")
	}
	if scientificName == "" {
		return nil, errors.New("scientific name is required")
	}

	return &Species{
		id:             id,
		scientificName: scientificName,
		commonName:     commonName,
		iconicTaxon:    iconicTaxon,
		photos:         make([]Photo, 0),
	}, nil
}

// ID returns the species ID.
func (s *Species) ID() int {
	return s.id
}

// ScientificName returns the scientific name.
func (s *Species) ScientificName() string {
	return s.scientificName
}

// CommonName returns the common name.
func (s *Species) CommonName() string {
	return s.commonName
}

// IconicTaxon returns the iconic taxon group.
func (s *Species) IconicTaxon() string {
	return s.iconicTaxon
}

// DisplayName returns the best display name available.
func (s *Species) DisplayName() string {
	if s.commonName != "" {
		return s.commonName
	}
	return s.scientificName
}

// Photos returns all photos for this species.
func (s *Species) Photos() []Photo {
	return s.photos
}

// AddPhoto adds a photo to the species.
func (s *Species) AddPhoto(photo Photo) {
	s.photos = append(s.photos, photo)
}

// HasPhotos checks if the species has any photos.
func (s *Species) HasPhotos() bool {
	return len(s.photos) > 0
}

// SetAncestorIDs sets the taxonomic ancestor IDs.
func (s *Species) SetAncestorIDs(ids []int) {
	s.ancestorIDs = ids
}

// AncestorIDs returns the taxonomic ancestor IDs.
func (s *Species) AncestorIDs() []int {
	return s.ancestorIDs
}

// SetRank sets the taxonomic rank.
func (s *Species) SetRank(rank string) {
	s.rank = rank
}

// Rank returns the taxonomic rank.
func (s *Species) Rank() string {
	return s.rank
}

// Snapshot is a serializable representation of a Species, used for
// persistence and reconstruction.
type Snapshot struct {
	ID             int     `json:"id"`
	ScientificName string  `json:"scientific_name"`
	CommonName     string  `json:"common_name"`
	IconicTaxon    string  `json:"iconic_taxon"`
	Rank           string  `json:"rank"`
	AncestorIDs    []int   `json:"ancestor_ids,omitempty"`
	Photos         []Photo `json:"photos,omitempty"`
}

// Snapshot returns a serializable representation of the species.
func (s *Species) Snapshot() Snapshot {
	return Snapshot{
		ID:             s.id,
		ScientificName: s.scientificName,
		CommonName:     s.commonName,
		IconicTaxon:    s.iconicTaxon,
		Rank:           s.rank,
		AncestorIDs:    s.ancestorIDs,
		Photos:         s.photos,
	}
}

// FromSnapshot reconstructs a Species from a snapshot.
func FromSnapshot(snap Snapshot) (*Species, error) {
	sp, err := New(snap.ID, snap.ScientificName, snap.CommonName, snap.IconicTaxon)
	if err != nil {
		return nil, err
	}
	sp.SetRank(snap.Rank)
	sp.SetAncestorIDs(snap.AncestorIDs)
	for _, p := range snap.Photos {
		sp.AddPhoto(p)
	}
	return sp, nil
}
