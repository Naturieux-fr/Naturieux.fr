// Package species contains domain entities for species and taxa.
package species

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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
	// Zoom, when set, is the area the Détail quiz mode crops to instead of a
	// random crop.
	Zoom *PhotoRegion
}

// PhotoRegion is a rectangular area of a photo expressed as fractions (0–1) of
// the image's width and height.
type PhotoRegion struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Species represents a biological species entity.
type Species struct {
	id             int
	scientificName string
	commonName     string
	iconicTaxon    string
	family         string
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

// MatchesName reports whether a free-text guess identifies this species. The
// comparison is tolerant of case, accents, surrounding articles and partial
// names (e.g. "renard" matches "Renard roux", "vulpes vulpes" matches the
// scientific name).
func (s *Species) MatchesName(guess string) bool {
	g := normalizeName(guess)
	if len(g) < 3 {
		return false
	}
	for _, candidate := range []string{s.scientificName, s.commonName} {
		c := normalizeName(candidate)
		if c == "" {
			continue
		}
		if c == g || strings.Contains(c, g) || strings.Contains(g, c) {
			return true
		}
	}
	return false
}

// normalizeName lowercases, strips accents and parentheticals, and keeps only
// letters and single spaces.
func normalizeName(s string) string {
	// Remove combining marks (accents).
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	folded, _, err := transform.String(t, s)
	if err != nil {
		folded = s
	}

	var b strings.Builder
	prevSpace := false
	for _, r := range strings.ToLower(folded) {
		switch {
		case unicode.IsLetter(r):
			b.WriteRune(r)
			prevSpace = false
		case (r == ' ' || r == '-' || r == '\'') && !prevSpace:
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
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

// GenusID returns the immediate parent taxon ID (usually the genus).
// iNaturalist ancestor lists may include the taxon itself as the last
// element; it is skipped. Returns 0 when no parent is known.
func (s *Species) GenusID() int {
	ancestors := s.ancestorIDs
	if len(ancestors) > 0 && ancestors[len(ancestors)-1] == s.id {
		ancestors = ancestors[:len(ancestors)-1]
	}
	if len(ancestors) == 0 {
		return 0
	}
	return ancestors[len(ancestors)-1]
}

// SetRank sets the taxonomic rank.
// Family returns the taxonomic family (may be empty).
func (s *Species) Family() string { return s.family }

// SetFamily sets the taxonomic family.
func (s *Species) SetFamily(family string) { s.family = family }

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
	Family         string  `json:"family,omitempty"`
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
		Family:         s.family,
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
	sp.SetFamily(snap.Family)
	sp.SetAncestorIDs(snap.AncestorIDs)
	for _, p := range snap.Photos {
		sp.AddPhoto(p)
	}
	return sp, nil
}
