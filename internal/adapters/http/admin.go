package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// AdminAuth authenticates administrators and validates session tokens.
type AdminAuth interface {
	Login(ctx context.Context, username, password string) (string, error)
	Authorize(ctx context.Context, token string) (string, error)
}

// MediaStore stores uploaded photo files.
type MediaStore interface {
	Save(ctx context.Context, contentType string, r io.Reader) (storage.Saved, error)
	Delete(ctx context.Context, url string) error
}

// maxUploadBytes caps an uploaded photo's size.
const maxUploadBytes = 10 << 20 // 10 MiB

// Inviter mints, lists and revokes player invitation tokens.
type Inviter interface {
	IssueInvite(ctx context.Context, createdBy string) (string, error)
	ListInvites(ctx context.Context) ([]ports.Invite, error)
	RevokeInvite(ctx context.Context, token string) error
}

// RoomStats reports the number of live multiplayer rooms.
type RoomStats interface {
	Count() int
}

// AdminHandler serves the back-office API. Photo management is only available
// when the species source is TAXREF (photos is non-nil).
type AdminHandler struct {
	auth       AdminAuth
	photos     *taxref.Repository        // nil when the species source has no managed photos
	storage    MediaStore                // nil when uploads are not configured
	inviter    Inviter                   // nil when account invites are unavailable
	players    ports.PlayerAdminStore    // nil when player admin is unavailable
	rooms      RoomStats                 // nil when room stats are unavailable
	curated    *sqlite.CuratedRepository // nil when curated quizzes are unavailable
	challenges *challenge.Manager        // nil when challenges are unavailable
}

// NewAdminHandler creates the admin API handler. photos, store and inviter may
// be nil.
func NewAdminHandler(auth AdminAuth, photos *taxref.Repository, store MediaStore, inviter Inviter) *AdminHandler {
	return &AdminHandler{auth: auth, photos: photos, storage: store, inviter: inviter}
}

// SetAdminData wires the dashboard and player-management data sources.
func (h *AdminHandler) SetAdminData(players ports.PlayerAdminStore, rooms RoomStats) {
	h.players = players
	h.rooms = rooms
}

// SetCuratedData wires the curated-quiz store and the challenge scheduler.
func (h *AdminHandler) SetCuratedData(curated *sqlite.CuratedRepository, challenges *challenge.Manager) {
	h.curated = curated
	h.challenges = challenges
}

// RegisterRoutes wires the admin endpoints onto the mux.
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("GET /api/v1/admin/taxa", h.requireAdmin(h.handleSearchTaxa))
	mux.HandleFunc("GET /api/v1/admin/taxa/{cd_nom}/photos", h.requireAdmin(h.handleListPhotos))
	mux.HandleFunc("POST /api/v1/admin/taxa/{cd_nom}/photos", h.requireAdmin(h.handleAddPhoto))
	mux.HandleFunc("POST /api/v1/admin/taxa/{cd_nom}/upload", h.requireAdmin(h.handleUploadPhoto))
	mux.HandleFunc("DELETE /api/v1/admin/photos/{id}", h.requireAdmin(h.handleDeletePhoto))
	mux.HandleFunc("POST /api/v1/admin/invites", h.requireAuth(h.handleCreateInvite))
	mux.HandleFunc("GET /api/v1/admin/invites", h.requireAuth(h.handleListInvites))
	mux.HandleFunc("POST /api/v1/admin/invites/{token}/revoke", h.requireAuth(h.handleRevokeInvite))
	mux.HandleFunc("GET /api/v1/admin/stats", h.requireAuth(h.handleStats))
	mux.HandleFunc("GET /api/v1/admin/players", h.requireAuth(h.handleListPlayers))
	mux.HandleFunc("GET /api/v1/admin/coverage", h.requireAuth(h.handleCoverage))
	mux.HandleFunc("GET /api/v1/admin/quizzes", h.requireAuth(h.handleListQuizzes))
	mux.HandleFunc("POST /api/v1/admin/quizzes", h.requireAuth(h.handleCreateQuiz))
	mux.HandleFunc("DELETE /api/v1/admin/quizzes/{id}", h.requireAuth(h.handleDeleteQuiz))
	mux.HandleFunc("POST /api/v1/admin/challenge/schedule", h.requireAuth(h.handleScheduleChallenge))
	mux.HandleFunc("DELETE /api/v1/admin/players/{id}", h.requireAuth(h.handleDeletePlayer))
	mux.HandleFunc("POST /api/v1/admin/players/{id}/role", h.requireAuth(h.handleSetPlayerRole))
}

// handleStats returns dashboard figures.
func (h *AdminHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	out := map[string]interface{}{}
	ctx := r.Context()
	if h.players != nil {
		if n, err := h.players.CountPlayers(ctx); err == nil {
			out["players"] = n
		}
		if n, err := h.players.TotalGames(ctx); err == nil {
			out["games"] = n
		}
	}
	if h.rooms != nil {
		out["rooms_active"] = h.rooms.Count()
	}
	if h.photos != nil {
		if n, err := h.photos.CountSpecies(ctx); err == nil {
			out["species"] = n
		}
		if n, err := h.photos.CountPhotos(ctx); err == nil {
			out["photos"] = n
		}
		if n, err := h.photos.CountSpeciesWithPhotos(ctx); err == nil {
			out["species_with_photos"] = n
		}
	}
	writeSuccess(w, out)
}

// handleListQuizzes returns the curated quizzes.
func (h *AdminHandler) handleListQuizzes(w http.ResponseWriter, r *http.Request) {
	if h.curated == nil {
		writeError(w, http.StatusServiceUnavailable, "curated quizzes require the TAXREF species source")
		return
	}
	list, err := h.curated.ListQuizzes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"quizzes": list})
}

// handleCreateQuiz stores a curated quiz (a named list of cd_noms).
func (h *AdminHandler) handleCreateQuiz(w http.ResponseWriter, r *http.Request) {
	if h.curated == nil {
		writeError(w, http.StatusServiceUnavailable, "curated quizzes require the TAXREF species source")
		return
	}
	var req struct {
		Name    string `json:"name"`
		Species []int  `json:"species"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" || len(req.Species) == 0 {
		writeError(w, http.StatusBadRequest, "name and at least one species are required")
		return
	}
	id := uuid.New().String()
	if err := h.curated.CreateQuiz(r.Context(), id, strings.TrimSpace(req.Name), req.Species, time.Now().UTC().Format(time.RFC3339)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"id": id})
}

// handleDeleteQuiz removes a curated quiz.
func (h *AdminHandler) handleDeleteQuiz(w http.ResponseWriter, r *http.Request) {
	if h.curated == nil {
		writeError(w, http.StatusServiceUnavailable, "curated quizzes unavailable")
		return
	}
	err := h.curated.DeleteQuiz(r.Context(), r.PathValue("id"))
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "quiz not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "deleted"})
}

// handleScheduleChallenge sets a curated quiz as the current daily/weekly défi.
func (h *AdminHandler) handleScheduleChallenge(w http.ResponseWriter, r *http.Request) {
	if h.curated == nil || h.challenges == nil {
		writeError(w, http.StatusServiceUnavailable, "challenges unavailable")
		return
	}
	var req struct {
		Period string `json:"period"`
		QuizID string `json:"quiz_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	period := challenge.Period(req.Period)
	if !period.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid period")
		return
	}
	key := h.challenges.Key(period)
	if err := h.curated.Schedule(r.Context(), string(period), key, req.QuizID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.challenges.Invalidate(period, key) // next player gets the curated set
	writeSuccess(w, map[string]string{"period": string(period), "key": key})
}

// handleCoverage returns the photo coverage per taxonomic group.
func (h *AdminHandler) handleCoverage(w http.ResponseWriter, r *http.Request) {
	if h.photos == nil {
		writeError(w, http.StatusServiceUnavailable, "coverage requires the TAXREF species source")
		return
	}
	rows, err := h.photos.PhotoCoverage(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"coverage": rows})
}

// handleListPlayers returns the player roster.
func (h *AdminHandler) handleListPlayers(w http.ResponseWriter, r *http.Request) {
	if h.players == nil {
		writeError(w, http.StatusServiceUnavailable, "player admin unavailable")
		return
	}
	list, err := h.players.ListPlayers(r.Context(), 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"players": list})
}

// handleDeletePlayer removes a player account.
func (h *AdminHandler) handleDeletePlayer(w http.ResponseWriter, r *http.Request) {
	if h.players == nil {
		writeError(w, http.StatusServiceUnavailable, "player admin unavailable")
		return
	}
	if h.wouldRemoveLastAdmin(r.Context(), r.PathValue("id")) {
		writeError(w, http.StatusConflict, "impossible de supprimer le dernier administrateur")
		return
	}
	err := h.players.DeletePlayer(r.Context(), r.PathValue("id"))
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "deleted"})
}

// handleSetPlayerRole promotes or demotes a player.
func (h *AdminHandler) handleSetPlayerRole(w http.ResponseWriter, r *http.Request) {
	if h.players == nil {
		writeError(w, http.StatusServiceUnavailable, "player admin unavailable")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Role == "player" && h.wouldRemoveLastAdmin(r.Context(), r.PathValue("id")) {
		writeError(w, http.StatusConflict, "impossible de rétrograder le dernier administrateur")
		return
	}
	err := h.players.SetRole(r.Context(), r.PathValue("id"), req.Role)
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "updated"})
}

// wouldRemoveLastAdmin reports whether removing or demoting the given player
// would leave the site with no administrator.
func (h *AdminHandler) wouldRemoveLastAdmin(ctx context.Context, id string) bool {
	role, err := h.players.Role(ctx, id)
	if err != nil || role != "admin" {
		return false
	}
	n, err := h.players.CountAdmins(ctx)
	return err == nil && n <= 1
}

// handleCreateInvite mints a player invitation token (the front-end turns it
// into a shareable link).
func (h *AdminHandler) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	if h.inviter == nil {
		writeError(w, http.StatusServiceUnavailable, "invitations unavailable")
		return
	}
	by, _ := h.auth.Authorize(r.Context(), bearerToken(r))
	token, err := h.inviter.IssueInvite(r.Context(), by)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"invite": token})
}

// handleListInvites returns the issued invitations.
func (h *AdminHandler) handleListInvites(w http.ResponseWriter, r *http.Request) {
	if h.inviter == nil {
		writeError(w, http.StatusServiceUnavailable, "invitations unavailable")
		return
	}
	list, err := h.inviter.ListInvites(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"invites": list})
}

// handleRevokeInvite disables an invitation link.
func (h *AdminHandler) handleRevokeInvite(w http.ResponseWriter, r *http.Request) {
	if h.inviter == nil {
		writeError(w, http.StatusServiceUnavailable, "invitations unavailable")
		return
	}
	err := h.inviter.RevokeInvite(r.Context(), r.PathValue("token"))
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "invitation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "revoked"})
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

// requireAuth wraps a handler, rejecting requests without a valid admin token.
func (h *AdminHandler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
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
		next(w, r)
	}
}

// requireAdmin is requireAuth plus a guard that photo management is available.
func (h *AdminHandler) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return h.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if h.photos == nil {
			writeError(w, http.StatusServiceUnavailable, "photo management requires the TAXREF species source")
			return
		}
		next(w, r)
	})
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

// handleUploadPhoto accepts a multipart upload, stores the image file and
// records the photo for the taxon. Form fields: file, attribution, license,
// difficulty.
func (h *AdminHandler) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		writeError(w, http.StatusServiceUnavailable, "uploads are not configured")
		return
	}
	cdNom, ok := pathInt(w, r, "cd_nom")
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+512)
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "a 'file' field is required")
		return
	}
	defer func() { _ = file.Close() }()

	difficulty := r.FormValue("difficulty")
	if difficulty != "" && !validPhotoDifficulty(difficulty) {
		writeError(w, http.StatusBadRequest, "invalid difficulty")
		return
	}

	// Sniff the real content type from the first bytes rather than trusting
	// the client-declared type or the file name.
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	contentType := http.DetectContentType(head[:n])
	if _, err := storage.ExtensionFor(contentType); err != nil {
		writeError(w, http.StatusUnsupportedMediaType, "only JPEG, PNG, WebP or GIF images are accepted")
		return
	}

	body := io.MultiReader(bytes.NewReader(head[:n]), file)
	saved, err := h.storage.Save(r.Context(), contentType, body)
	if errors.Is(err, storage.ErrTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	id, err := h.photos.AddPhoto(r.Context(), cdNom, saved.URL, r.FormValue("attribution"), r.FormValue("license"), difficulty)
	if errors.Is(err, ports.ErrNotFound) {
		_ = h.storage.Delete(r.Context(), saved.URL) // don't keep an orphan file
		writeError(w, http.StatusNotFound, "taxon not found")
		return
	}
	if err != nil {
		_ = h.storage.Delete(r.Context(), saved.URL)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"id": id, "url": saved.URL})
}

// handleDeletePhoto removes a photo by id.
func (h *AdminHandler) handleDeletePhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	url, err := h.photos.DeletePhoto(r.Context(), id)
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "photo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Remove the stored file too (no-op for external URLs).
	if h.storage != nil {
		if delErr := h.storage.Delete(r.Context(), url); delErr != nil {
			log.Printf("admin: removing stored file %q: %v", url, delErr)
		}
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
