package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
)

// AccountHandler serves player registration and login.
type AccountHandler struct {
	accounts *account.Service
}

// NewAccountHandler creates the account API handler.
func NewAccountHandler(accounts *account.Service) *AccountHandler {
	return &AccountHandler{accounts: accounts}
}

// RegisterRoutes wires the account endpoints.
func (h *AccountHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/account/register", h.handleRegister)
	mux.HandleFunc("POST /api/v1/account/login", h.handleLogin)
	mux.HandleFunc("GET /api/v1/account/me", h.handleMe)
	mux.HandleFunc("POST /api/v1/account/reset", h.handleReset)
}

// handleReset consumes an admin-issued reset token and sets a new password.
func (h *AccountHandler) handleReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	player, token, err := h.accounts.ResetPassword(r.Context(), req.Token, req.Password)
	if err != nil {
		h.writeAccountError(w, err)
		return
	}
	writeSuccess(w, map[string]interface{}{"token": token, "player": playerToDTO(player)})
}

// handleMe returns the authenticated player's id, username and role.
func (h *AccountHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	id, username, role, err := h.accounts.Me(r.Context(), bearerToken(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentification requise")
		return
	}
	writeSuccess(w, map[string]string{"id": id, "username": username, "role": role})
}

func (h *AccountHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Invite   string `json:"invite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	player, token, err := h.accounts.Register(r.Context(), req.Username, req.Password, req.Invite)
	if err != nil {
		h.writeAccountError(w, err)
		return
	}
	dto := playerToDTO(player)
	writeSuccess(w, map[string]interface{}{"token": token, "player": dto})
}

func (h *AccountHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	player, token, err := h.accounts.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		h.writeAccountError(w, err)
		return
	}
	dto := playerToDTO(player)
	writeSuccess(w, map[string]interface{}{"token": token, "player": dto})
}

func (h *AccountHandler) writeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, account.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "identifiants incorrects")
	case errors.Is(err, account.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "ce nom est déjà pris")
	case errors.Is(err, account.ErrInviteRequired):
		writeError(w, http.StatusForbidden, "inscription sur invitation uniquement")
	case errors.Is(err, account.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, "mot de passe trop court (6 caractères minimum)")
	case errors.Is(err, account.ErrBadUsername):
		writeError(w, http.StatusBadRequest, "nom invalide (2 à 20 caractères)")
	case errors.Is(err, account.ErrInvalidReset):
		writeError(w, http.StatusForbidden, "lien de réinitialisation invalide ou expiré")
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
