package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// LocateHandler serves the "Où est l'espèce ?" exercise: the player is shown a
// multi-species photo and must click the region containing a named species.
// Zone positions are never sent before answering, and the click is validated
// server-side against the photo's stored zones.
type LocateHandler struct {
	photos *taxref.Repository
	images *Handler // reused for the opaque image proxy
	pick   func(n int) int
}

// NewLocateHandler creates the locate exercise handler.
func NewLocateHandler(photos *taxref.Repository, images *Handler) *LocateHandler {
	return &LocateHandler{photos: photos, images: images, pick: func(n int) int {
		if n <= 0 {
			return 0
		}
		return rand.Intn(n) // #nosec G404 -- picking a quiz item, not security-sensitive
	}}
}

// RegisterRoutes wires the locate endpoints.
func (h *LocateHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/locate/next", h.handleNext)
	mux.HandleFunc("GET /api/v1/locate/image/{id}", h.handleImage)
	mux.HandleFunc("POST /api/v1/locate/answer", h.handleAnswer)
}

func (h *LocateHandler) handleNext(w http.ResponseWriter, r *http.Request) {
	photos, err := h.photos.PhotosWithSpeciesZones(r.Context(), 30)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(photos) == 0 {
		writeError(w, http.StatusNotFound, "aucune photo annotée pour le moment")
		return
	}
	p := photos[h.pick(len(photos))]
	target := p.Species[h.pick(len(p.Species))]

	// Names of every species present, so the prompt can list the options.
	names := make([]string, 0, len(p.Species))
	for _, s := range p.Species {
		names = append(names, s.Name)
	}
	writeSuccess(w, map[string]interface{}{
		"photo_id":     p.PhotoID,
		"image_url":    fmt.Sprintf("/api/v1/locate/image/%d", p.PhotoID),
		"target_cd":    target.CdNom,
		"target_name":  target.Name,
		"species_here": names,
	})
}

func (h *LocateHandler) handleImage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	url, err := h.photos.PhotoURLByID(r.Context(), id)
	if errors.Is(err, ports.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.images.streamImage(w, r, url)
}

func (h *LocateHandler) handleAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PhotoID int     `json:"photo_id"`
		CdNom   int     `json:"cd_nom"`
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	zones, err := h.photos.PhotoSpeciesZones(r.Context(), req.PhotoID)
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "photo introuvable")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	correct := false
	for _, z := range zones {
		if z.CdNom == req.CdNom && req.X >= z.X && req.X <= z.X+z.W && req.Y >= z.Y && req.Y <= z.Y+z.H {
			correct = true
			break
		}
	}
	// Reveal all zones so the client can show where each species was.
	writeSuccess(w, map[string]interface{}{
		"correct":   correct,
		"target_cd": req.CdNom,
		"zones":     zones,
	})
}
