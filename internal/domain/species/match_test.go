package species_test

import (
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

func TestSpecies_MatchesName(t *testing.T) {
	sp, _ := species.New(60585, "Vulpes vulpes", "Renard roux", "Mammifères")

	matches := []string{
		"Renard roux", "renard roux", "  RENARD  ROUX ",
		"renard", "le renard roux", // partial / article
		"Vulpes vulpes", "vulpes vulpes",
	}
	for _, g := range matches {
		if !sp.MatchesName(g) {
			t.Errorf("MatchesName(%q) = false, want true", g)
		}
	}

	rejects := []string{"", "ab", "loup", "blaireau", "vipere"}
	for _, g := range rejects {
		if sp.MatchesName(g) {
			t.Errorf("MatchesName(%q) = true, want false", g)
		}
	}
}

func TestSpecies_MatchesName_AccentInsensitive(t *testing.T) {
	sp, _ := species.New(1, "Bufo bufo", "Crapaud épineux", "Amphibiens")
	for _, g := range []string{"crapaud epineux", "Crapaud Épineux", "épineux"} {
		if !sp.MatchesName(g) {
			t.Errorf("MatchesName(%q) = false, want true (accent-insensitive)", g)
		}
	}
}
