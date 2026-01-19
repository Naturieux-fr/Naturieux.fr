package inaturalist_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/inaturalist"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

func TestNewClient(t *testing.T) {
	client := inaturalist.NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	customHTTP := &http.Client{}
	client := inaturalist.NewClient(
		inaturalist.WithHTTPClient(customHTTP),
		inaturalist.WithUserAgent("TestAgent/1.0"),
		inaturalist.WithBaseURL("https://test.example.com"),
	)
	if client == nil {
		t.Fatal("NewClient() with options returned nil")
	}
}

func TestClient_GetByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/taxa" {
			t.Errorf("Expected path /taxa, got %s", r.URL.Path)
		}

		response := map[string]interface{}{
			"total_results": 1,
			"results": []map[string]interface{}{
				{
					"id":                    42069,
					"name":                  "Vulpes vulpes",
					"rank":                  "species",
					"preferred_common_name": "Red Fox",
					"iconic_taxon_name":     "Mammalia",
					"ancestor_ids":          []int{1, 2, 3, 4},
					"default_photo": map[string]interface{}{
						"id":         123,
						"medium_url": "https://example.com/photo.jpg",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	sp, err := client.GetByID(context.Background(), 42069)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if sp.ID() != 42069 {
		t.Errorf("Species ID = %d, want 42069", sp.ID())
	}
	if sp.ScientificName() != "Vulpes vulpes" {
		t.Errorf("ScientificName = %s, want Vulpes vulpes", sp.ScientificName())
	}
	if sp.CommonName() != "Red Fox" {
		t.Errorf("CommonName = %s, want Red Fox", sp.CommonName())
	}
	if sp.IconicTaxon() != "Mammalia" {
		t.Errorf("IconicTaxon = %s, want Mammalia", sp.IconicTaxon())
	}
}

func TestClient_GetByID_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"total_results": 0,
			"results":       []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	_, err := client.GetByID(context.Background(), 999999)
	if err == nil {
		t.Error("GetByID() should return error for non-existent species")
	}
}

func TestClient_GetRandom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/observations" {
			t.Errorf("Expected path /observations, got %s", r.URL.Path)
		}

		// Check query parameters
		query := r.URL.Query()
		if query.Get("photos") != "true" {
			t.Error("Expected photos=true")
		}
		if query.Get("order_by") != "random" {
			t.Error("Expected order_by=random")
		}

		response := map[string]interface{}{
			"total_results": 2,
			"results": []map[string]interface{}{
				{
					"id": 1,
					"taxon": map[string]interface{}{
						"id":                    100,
						"name":                  "Species 1",
						"preferred_common_name": "Common 1",
						"iconic_taxon_name":     "Mammalia",
					},
					"photos": []map[string]interface{}{
						{"id": 1, "medium_url": "https://example.com/1.jpg"},
					},
				},
				{
					"id": 2,
					"taxon": map[string]interface{}{
						"id":                    200,
						"name":                  "Species 2",
						"preferred_common_name": "Common 2",
						"iconic_taxon_name":     "Aves",
					},
					"photos": []map[string]interface{}{
						{"id": 2, "medium_url": "https://example.com/2.jpg"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	filter := ports.SpeciesFilter{
		IconicTaxon: "Mammalia",
		Limit:       10,
	}

	species, err := client.GetRandom(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	if len(species) != 2 {
		t.Errorf("GetRandom() returned %d species, want 2", len(species))
	}

	// Check that photos were added
	for _, sp := range species {
		if !sp.HasPhotos() {
			t.Errorf("Species %s should have photos", sp.ScientificName())
		}
	}
}

func TestClient_GetRandom_Deduplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return duplicate species
		response := map[string]interface{}{
			"total_results": 3,
			"results": []map[string]interface{}{
				{
					"id": 1,
					"taxon": map[string]interface{}{
						"id":   100,
						"name": "Species 1",
					},
					"photos": []map[string]interface{}{
						{"id": 1, "medium_url": "https://example.com/1.jpg"},
					},
				},
				{
					"id": 2,
					"taxon": map[string]interface{}{
						"id":   100, // Same taxon ID
						"name": "Species 1",
					},
					"photos": []map[string]interface{}{
						{"id": 2, "medium_url": "https://example.com/2.jpg"},
					},
				},
				{
					"id": 3,
					"taxon": map[string]interface{}{
						"id":   200,
						"name": "Species 2",
					},
					"photos": []map[string]interface{}{
						{"id": 3, "medium_url": "https://example.com/3.jpg"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	species, err := client.GetRandom(context.Background(), ports.SpeciesFilter{})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	// Should have 2 unique species, not 3
	if len(species) != 2 {
		t.Errorf("GetRandom() returned %d species, want 2 (deduplicated)", len(species))
	}
}

func TestClient_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/taxa/autocomplete" {
			t.Errorf("Expected path /taxa/autocomplete, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("q") != "vulpes" {
			t.Errorf("Expected q=vulpes, got %s", query.Get("q"))
		}

		response := map[string]interface{}{
			"total_results": 2,
			"results": []map[string]interface{}{
				{
					"id":                    1,
					"name":                  "Vulpes vulpes",
					"preferred_common_name": "Red Fox",
				},
				{
					"id":                    2,
					"name":                  "Vulpes zerda",
					"preferred_common_name": "Fennec Fox",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	species, err := client.Search(context.Background(), "vulpes", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(species) != 2 {
		t.Errorf("Search() returned %d species, want 2", len(species))
	}
}

func TestClient_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	_, err := client.GetByID(context.Background(), 1)
	if err == nil {
		t.Error("GetByID() should return error on HTTP 500")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetByID(ctx, 1)
	if err == nil {
		t.Error("GetByID() should return error on canceled context")
	}
}

func TestClient_GetSimilar(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount == 1 {
			// First request: GetByID to find the species
			response := map[string]interface{}{
				"total_results": 1,
				"results": []map[string]interface{}{
					{
						"id":           42069,
						"name":         "Vulpes vulpes",
						"rank":         "species",
						"ancestor_ids": []int{1, 2, 3, 4, 5, 6},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Second request: Get similar species from genus
		response := map[string]interface{}{
			"total_results": 3,
			"results": []map[string]interface{}{
				{
					"id":   42069, // Same species (should be excluded)
					"name": "Vulpes vulpes",
					"rank": "species",
				},
				{
					"id":   42070,
					"name": "Vulpes zerda",
					"rank": "species",
				},
				{
					"id":   42071,
					"name": "Vulpes lagopus",
					"rank": "species",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	similar, err := client.GetSimilar(context.Background(), 42069, 2)
	if err != nil {
		t.Fatalf("GetSimilar() error = %v", err)
	}

	// Should have 2 similar species (excluding the original)
	if len(similar) != 2 {
		t.Errorf("GetSimilar() returned %d species, want 2", len(similar))
	}

	// Verify none of them is the original species
	for _, sp := range similar {
		if sp.ID() == 42069 {
			t.Error("GetSimilar() should not include the original species")
		}
	}
}

func TestClient_GetSimilar_NotEnoughAncestors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"total_results": 1,
			"results": []map[string]interface{}{
				{
					"id":           1,
					"name":         "Test Species",
					"ancestor_ids": []int{1}, // Not enough ancestors
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	_, err := client.GetSimilar(context.Background(), 1, 2)
	if err == nil {
		t.Error("GetSimilar() should return error when species has not enough ancestors")
	}
}

func TestClient_GetRandom_WithExcludeIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("without_taxon_id") == "" {
			t.Error("Expected without_taxon_id parameter")
		}

		response := map[string]interface{}{
			"total_results": 1,
			"results": []map[string]interface{}{
				{
					"id": 1,
					"taxon": map[string]interface{}{
						"id":   300,
						"name": "New Species",
					},
					"photos": []map[string]interface{}{
						{"id": 1, "medium_url": "https://example.com/1.jpg"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	filter := ports.SpeciesFilter{
		ExcludeIDs: []int{100, 200},
	}

	_, err := client.GetRandom(context.Background(), filter)
	if err != nil {
		t.Fatalf("GetRandom() with exclude IDs error = %v", err)
	}
}

func TestClient_GetRandom_NilTaxon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"total_results": 2,
			"results": []map[string]interface{}{
				{
					"id":    1,
					"taxon": nil, // Nil taxon should be skipped
				},
				{
					"id": 2,
					"taxon": map[string]interface{}{
						"id":   100,
						"name": "Valid Species",
					},
					"photos": []map[string]interface{}{
						{"id": 1, "medium_url": "https://example.com/1.jpg"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := inaturalist.NewClient(
		inaturalist.WithBaseURL(server.URL),
	)

	species, err := client.GetRandom(context.Background(), ports.SpeciesFilter{})
	if err != nil {
		t.Fatalf("GetRandom() error = %v", err)
	}

	// Should only have 1 species (nil taxon skipped)
	if len(species) != 1 {
		t.Errorf("GetRandom() returned %d species, want 1 (nil taxon skipped)", len(species))
	}
}
