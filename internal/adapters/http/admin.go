package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// AdminAuth authenticates administrators and validates session tokens.
type AdminAuth interface {
	Login(ctx context.Context, username, password string) (string, error)
	Authorize(ctx context.Context, token string) (string, error)
}

// AdminHandler serves the back-office API. Photo management is only available
// when the species source is TAXREF (photos is non-nil).
type AdminHandler struct {
	auth   AdminAuth
	photos *taxref.Repository // nil when the species source has no managed photos
}

// NewAdminHandler creates the admin API handler. photos may be nil.
func NewAdminHandler(auth AdminAuth, photos *taxref.Repository) *AdminHandler {
	return &AdminHandler{auth: auth, photos: photos}
}

// RegisterRoutes wires the admin endpoints onto the mux.
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("GET /api/v1/admin/taxa", h.requireAdmin(h.handleSearchTaxa))
	mux.HandleFunc("GET /api/v1/admin/taxa/{cd_nom}/photos", h.requireAdmin(h.handleListPhotos))
	mux.HandleFunc("POST /api/v1/admin/taxa/{cd_nom}/photos", h.requireAdmin(h.handleAddPhoto))
	mux.HandleFunc("DELETE /api/v1/admin/photos/{id}", h.requireAdmin(h.handleDeletePhoto))
}

// handleLogin authenticates an admin and returns a session token.
func (h *AdminHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	token, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeSuccess(w, map[string]string{"token": token})
}

// requireAdmin wraps a handler, rejecting requests without a valid admin token.
func (h *AdminHandler) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		if _, err := h.auth.Authorize(r.Context(), token); err != nil {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		if h.photos == nil {
			writeError(w, http.StatusServiceUnavailable, "photo management requires the TAXREF species source")
			return
		}
		next(w, r)
	}
}

// handleSearchTaxa finds taxa by name (scientific or vernacular).
func (h *AdminHandler) handleSearchTaxa(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "query 'q' is required")
		return
	}
	results, err := h.photos.Search(r.Context(), q, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	taxa := make([]map[string]interface{}, len(results))
	for i, sp := range results {
		taxa[i] = map[string]interface{}{
			"cd_nom":          sp.ID(),
			"scientific_name": sp.ScientificName(),
			"vernacular_name": sp.CommonName(),
			"iconic_taxon":    sp.IconicTaxon(),
		}
	}
	writeSuccess(w, map[string]interface{}{"taxa": taxa})
}

// handleListPhotos returns the owned photos of a taxon.
func (h *AdminHandler) handleListPhotos(w http.ResponseWriter, r *http.Request) {
	cdNom, ok := pathInt(w, r, "cd_nom")
	if !ok {
		return
	}
	photos, err := h.photos.ListPhotos(r.Context(), cdNom)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"photos": photos})
}

// handleAddPhoto attaches a photo to a taxon.
func (h *AdminHandler) handleAddPhoto(w http.ResponseWriter, r *http.Request) {
	cdNom, ok := pathInt(w, r, "cd_nom")
	if !ok {
		return
	}
	var req struct {
		URL         string `json:"url"`
		Attribution string `json:"attribution"`
		License     string `json:"license"`
		Difficulty  string `json:"difficulty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.Difficulty != "" && !validPhotoDifficulty(req.Difficulty) {
		writeError(w, http.StatusBadRequest, "invalid difficulty")
		return
	}

	id, err := h.photos.AddPhoto(r.Context(), cdNom, req.URL, req.Attribution, req.License, req.Difficulty)
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "taxon not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"id": id})
}

// handleDeletePhoto removes a photo by id.
func (h *AdminHandler) handleDeletePhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	err := h.photos.DeletePhoto(r.Context(), id)
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "photo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "deleted"})
}

// validPhotoDifficulty reports whether d is one of the quiz difficulty levels.
func validPhotoDifficulty(d string) bool {
	switch d {
	case "beginner", "intermediate", "expert", "master":
		return true
	}
	return false
}

// bearerToken extracts the token from the Authorization header.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// pathInt reads an integer path value, writing a 400 on failure.
func pathInt(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	v, err := strconv.Atoi(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, name+" must be an integer")
		return 0, false
	}
	return v, true
}
