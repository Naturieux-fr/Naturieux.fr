package mock_test

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

func TestMockSpeciesRepository(t *testing.T) {
	r := mock.NewSpeciesRepository()
	ctx := context.Background()

	got, err := r.GetRandom(ctx, ports.SpeciesFilter{Limit: 3, HasPhotos: true})
	if err != nil || len(got) == 0 {
		t.Fatalf("GetRandom = %d, %v", len(got), err)
	}
	id := got[0].ID()

	if sp, err := r.GetByID(ctx, id); err != nil || sp == nil {
		t.Errorf("GetByID = %v, %v", sp, err)
	}
	if sim, err := r.GetSimilar(ctx, id, 3); err != nil {
		t.Errorf("GetSimilar = %d, %v", len(sim), err)
	}
	if res, err := r.Search(ctx, got[0].ScientificName(), 5); err != nil || len(res) == 0 {
		t.Errorf("Search = %d, %v", len(res), err)
	}
	r.ResetUsed()

	// A category filter narrows the pool.
	if _, err := r.GetRandom(ctx, ports.SpeciesFilter{IconicTaxon: "Aves", Limit: 1}); err != nil {
		t.Errorf("GetRandom filtered: %v", err)
	}
}
