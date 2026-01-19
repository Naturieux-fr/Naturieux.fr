// Package inaturalist provides a client for the iNaturalist API.
package inaturalist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

const (
	defaultBaseURL   = "https://api.inaturalist.org/v1"
	defaultUserAgent = "Naturieux/1.0 (https://naturieux.fr)"
	defaultTimeout   = 10 * time.Second
)

// Client is an iNaturalist API client.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	userAgent   string
	rateLimiter *rateLimiter
}

// ClientOption configures the client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithUserAgent sets a custom user agent.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// NewClient creates a new iNaturalist client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		userAgent:   defaultUserAgent,
		rateLimiter: newRateLimiter(time.Second),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// rateLimiter ensures we respect rate limits.
type rateLimiter struct {
	interval time.Duration
	lastCall time.Time
}

func newRateLimiter(interval time.Duration) *rateLimiter {
	return &rateLimiter{interval: interval}
}

func (r *rateLimiter) wait() {
	elapsed := time.Since(r.lastCall)
	if elapsed < r.interval {
		time.Sleep(r.interval - elapsed)
	}
	r.lastCall = time.Now()
}

// API Response structures

type observationsResponse struct {
	TotalResults int           `json:"total_results"`
	Results      []observation `json:"results"`
}

type observation struct {
	ID           int     `json:"id"`
	SpeciesGuess string  `json:"species_guess"`
	Taxon        *taxon  `json:"taxon"`
	Photos       []photo `json:"photos"`
	Location     string  `json:"location"`
	PlaceGuess   string  `json:"place_guess"`
}

type taxon struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Rank                string `json:"rank"`
	PreferredCommonName string `json:"preferred_common_name"`
	IconicTaxonName     string `json:"iconic_taxon_name"`
	AncestorIDs         []int  `json:"ancestor_ids"`
	DefaultPhoto        *photo `json:"default_photo"`
}

type photo struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	MediumURL   string `json:"medium_url"`
	LargeURL    string `json:"large_url"`
	OriginalURL string `json:"original_url"`
	SquareURL   string `json:"square_url"`
	Attribution string `json:"attribution"`
}

type taxaResponse struct {
	TotalResults int     `json:"total_results"`
	Results      []taxon `json:"results"`
}

// doRequest performs an HTTP request with rate limiting.
func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values) (*http.Response, error) {
	c.rateLimiter.wait()

	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if len(params) > 0 {
		reqURL = fmt.Sprintf("%s?%s", reqURL, params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close() // Error ignored: we're already returning an error
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, nil
}

// GetByID retrieves a species by its iNaturalist taxon ID.
func (c *Client) GetByID(ctx context.Context, id int) (*species.Species, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(id))

	resp, err := c.doRequest(ctx, "/taxa", params)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result taxaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("species not found: %d", id)
	}

	return taxonToSpecies(&result.Results[0]), nil
}

// Default values for API requests.
const (
	defaultPerPage = "20"
	maxPerPage     = 200
)

// GetRandom retrieves random species with photos matching the filter.
func (c *Client) GetRandom(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	params := c.buildObservationParams(filter)

	resp, err := c.doRequest(ctx, "/observations", params)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result observationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return c.deduplicateObservations(result.Results), nil
}

// buildObservationParams creates URL parameters for observation queries.
func (c *Client) buildObservationParams(filter ports.SpeciesFilter) url.Values {
	params := url.Values{}
	params.Set("photos", "true")
	params.Set("quality_grade", "research")
	params.Set("identified", "true")
	params.Set("order_by", "random")

	c.applyFilterParams(params, filter)
	return params
}

// applyFilterParams applies filter options to URL parameters.
func (c *Client) applyFilterParams(params url.Values, filter ports.SpeciesFilter) {
	if filter.Limit > 0 {
		params.Set("per_page", strconv.Itoa(min(filter.Limit, maxPerPage)))
	} else {
		params.Set("per_page", defaultPerPage)
	}

	if filter.IconicTaxon != "" {
		params.Set("iconic_taxa", filter.IconicTaxon)
	}

	if filter.PlaceID > 0 {
		params.Set("place_id", strconv.Itoa(filter.PlaceID))
	}

	if len(filter.ExcludeIDs) > 0 {
		params.Set("without_taxon_id", c.formatIDList(filter.ExcludeIDs))
	}
}

// formatIDList converts a slice of IDs to a comma-separated string.
func (c *Client) formatIDList(ids []int) string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.Itoa(id)
	}
	return strings.Join(strs, ",")
}

// deduplicateObservations extracts unique species from observations.
func (c *Client) deduplicateObservations(observations []observation) []*species.Species {
	seen := make(map[int]bool)
	speciesList := make([]*species.Species, 0, len(observations))

	for _, obs := range observations {
		if obs.Taxon == nil || seen[obs.Taxon.ID] {
			continue
		}
		seen[obs.Taxon.ID] = true
		speciesList = append(speciesList, c.observationToSpecies(obs))
	}

	return speciesList
}

// observationToSpecies converts an observation to a species with photos.
func (c *Client) observationToSpecies(obs observation) *species.Species {
	sp := taxonToSpecies(obs.Taxon)
	for _, p := range obs.Photos {
		sp.AddPhoto(photoToSpeciesPhoto(&p))
	}
	return sp
}

// GetSimilar retrieves species in the same genus or family.
func (c *Client) GetSimilar(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
	// First, get the species to find its ancestors
	sp, err := c.GetByID(ctx, speciesID)
	if err != nil {
		return nil, err
	}

	ancestors := sp.AncestorIDs()
	if len(ancestors) < 2 {
		return nil, fmt.Errorf("not enough ancestor data for species %d", speciesID)
	}

	// Use the genus (second to last ancestor typically)
	genusID := ancestors[len(ancestors)-1]

	params := url.Values{}
	params.Set("taxon_id", strconv.Itoa(genusID))
	params.Set("rank", "species")
	params.Set("per_page", strconv.Itoa(min(limit+1, 30))) // +1 to account for excluding target

	resp, err := c.doRequest(ctx, "/taxa", params)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result taxaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	speciesList := make([]*species.Species, 0, limit)
	for _, t := range result.Results {
		if t.ID == speciesID {
			continue // Skip the target species
		}
		speciesList = append(speciesList, taxonToSpecies(&t))
		if len(speciesList) >= limit {
			break
		}
	}

	return speciesList, nil
}

// Search searches for species by name.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]*species.Species, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("rank", "species")
	params.Set("is_active", "true")
	params.Set("per_page", strconv.Itoa(min(limit, 30)))

	resp, err := c.doRequest(ctx, "/taxa/autocomplete", params)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result taxaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	speciesList := make([]*species.Species, 0, len(result.Results))
	for _, t := range result.Results {
		speciesList = append(speciesList, taxonToSpecies(&t))
	}

	return speciesList, nil
}

// taxonToSpecies converts an API taxon to a domain Species.
func taxonToSpecies(t *taxon) *species.Species {
	sp, _ := species.New(t.ID, t.Name, t.PreferredCommonName, t.IconicTaxonName)
	sp.SetAncestorIDs(t.AncestorIDs)
	sp.SetRank(t.Rank)

	if t.DefaultPhoto != nil {
		sp.AddPhoto(photoToSpeciesPhoto(t.DefaultPhoto))
	}

	return sp
}

// photoToSpeciesPhoto converts an API photo to a domain Photo.
func photoToSpeciesPhoto(p *photo) species.Photo {
	return species.Photo{
		ID:          p.ID,
		URL:         p.URL,
		MediumURL:   p.MediumURL,
		LargeURL:    p.LargeURL,
		OriginalURL: p.OriginalURL,
		SquareURL:   p.SquareURL,
		Attribution: p.Attribution,
	}
}
