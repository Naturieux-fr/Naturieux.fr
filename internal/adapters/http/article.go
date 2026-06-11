package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// Identifier resolves a session token to a player's id, username and role.
type Identifier interface {
	Me(ctx context.Context, token string) (id, username, role string, err error)
}

// ArticleHandler serves the learning library: public reading plus authoring by
// rédacteurs (role "writer") and admins.
type ArticleHandler struct {
	repo     *sqlite.ArticleRepository
	identity Identifier
}

// NewArticleHandler creates the article API handler.
func NewArticleHandler(repo *sqlite.ArticleRepository, identity Identifier) *ArticleHandler {
	return &ArticleHandler{repo: repo, identity: identity}
}

// RegisterRoutes wires the article endpoints.
func (h *ArticleHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/articles", h.handleList)
	mux.HandleFunc("GET /api/v1/articles/by-species/{cd_nom}", h.handleBySpecies)
	mux.HandleFunc("GET /api/v1/articles/{id}", h.handleGet)
	mux.HandleFunc("POST /api/v1/articles", h.handleSave)
	mux.HandleFunc("PUT /api/v1/articles/{id}", h.handleSave)
	mux.HandleFunc("DELETE /api/v1/articles/{id}", h.handleDelete)
}

// author authorizes a writer/admin and returns their id and name.
func (h *ArticleHandler) author(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	id, name, role, err := h.identity.Me(r.Context(), bearerToken(r))
	if err != nil || id == "" {
		writeError(w, http.StatusUnauthorized, "authentification requise")
		return "", "", false
	}
	if role != "writer" && role != "admin" {
		writeError(w, http.StatusForbidden, "réservé aux rédacteurs")
		return "", "", false
	}
	return id, name, true
}

func (h *ArticleHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Writers/admins see drafts too; the public sees only published articles.
	publishedOnly := true
	if _, _, role, err := h.identity.Me(r.Context(), bearerToken(r)); err == nil && (role == "writer" || role == "admin") {
		publishedOnly = false
	}
	list, err := h.repo.List(r.Context(), publishedOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"articles": list})
}

func (h *ArticleHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	a, err := h.repo.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "article introuvable")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !a.Published {
		// Drafts are only visible to writers/admins.
		if _, _, role, err := h.identity.Me(r.Context(), bearerToken(r)); err != nil || (role != "writer" && role != "admin") {
			writeError(w, http.StatusNotFound, "article introuvable")
			return
		}
	}
	writeSuccess(w, map[string]interface{}{"article": a})
}

func (h *ArticleHandler) handleBySpecies(w http.ResponseWriter, r *http.Request) {
	cdNom, err := strconv.Atoi(r.PathValue("cd_nom"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cd_nom")
		return
	}
	list, err := h.repo.BySpecies(r.Context(), cdNom)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"articles": list})
}

func (h *ArticleHandler) handleSave(w http.ResponseWriter, r *http.Request) {
	authorID, authorName, ok := h.author(w, r)
	if !ok {
		return
	}
	var req struct {
		Title     string                  `json:"title"`
		Body      string                  `json:"body"`
		Species   []sqlite.ArticleSpecies `json:"species"`
		Published *bool                   `json:"published"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		writeError(w, http.StatusBadRequest, "titre et contenu requis")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id := r.PathValue("id")
	a := sqlite.Article{
		Title: strings.TrimSpace(req.Title), Body: req.Body, Species: req.Species,
		AuthorID: authorID, AuthorName: authorName, Published: true, UpdatedAt: now,
	}
	if req.Published != nil {
		a.Published = *req.Published
	}

	if id == "" { // create
		a.ID = uuid.New().String()
		a.CreatedAt = now
	} else { // update: preserve creation metadata
		existing, err := h.repo.Get(r.Context(), id)
		if errors.Is(err, ports.ErrNotFound) {
			writeError(w, http.StatusNotFound, "article introuvable")
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.ID = id
		a.CreatedAt = existing.CreatedAt
		a.AuthorID = existing.AuthorID
		a.AuthorName = existing.AuthorName
	}
	if err := h.repo.Save(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]interface{}{"id": a.ID})
}

func (h *ArticleHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.author(w, r); !ok {
		return
	}
	err := h.repo.Delete(r.Context(), r.PathValue("id"))
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "article introuvable")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"message": "deleted"})
}
