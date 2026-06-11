package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// ChallengeHandler serves the daily/weekly challenges. A challenge run is an
// ordinary quiz session seeded with the period's shared questions; its final
// score (read server-side from the session) is recorded to a leaderboard.
type ChallengeHandler struct {
	mgr     *challenge.Manager
	quiz    *appquiz.Service
	scores  *sqlite.ChallengeRepository
	players ports.PlayerRepository
	auth    Authenticator
}

// NewChallengeHandler creates the challenge API handler.
func NewChallengeHandler(mgr *challenge.Manager, quiz *appquiz.Service, scores *sqlite.ChallengeRepository, players ports.PlayerRepository) *ChallengeHandler {
	return &ChallengeHandler{mgr: mgr, quiz: quiz, scores: scores, players: players}
}

// SetAuthenticator enables player-token auth on challenge endpoints.
func (h *ChallengeHandler) SetAuthenticator(a Authenticator) { h.auth = a }

// RegisterRoutes wires the challenge endpoints.
func (h *ChallengeHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/challenge/{period}", h.handleState)
	mux.HandleFunc("POST /api/v1/challenge/{period}/start", h.handleStart)
	mux.HandleFunc("POST /api/v1/challenge/finish", h.handleFinish)
}

func (h *ChallengeHandler) handleState(w http.ResponseWriter, r *http.Request) {
	period := challenge.Period(r.PathValue("period"))
	if !period.IsValid() {
		writeError(w, http.StatusNotFound, "unknown challenge")
		return
	}
	key := h.mgr.Key(period)
	board, err := h.scores.Leaderboard(r.Context(), string(period), key, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]interface{}{"period": period, "key": key, "leaderboard": board}
	if pid := r.URL.Query().Get("player_id"); pid != "" {
		if best, played, err := h.scores.BestFor(r.Context(), string(period), key, pid); err == nil {
			out["your_best"] = best
			out["played"] = played
		}
	}
	writeSuccess(w, out)
}

func (h *ChallengeHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	period := challenge.Period(r.PathValue("period"))
	if !period.IsValid() {
		writeError(w, http.StatusNotFound, "unknown challenge")
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if h.auth != nil {
		pid, err := h.auth.Authenticate(bearerToken(r))
		if err != nil || pid == "" {
			writeError(w, http.StatusUnauthorized, "authentification requise")
			return
		}
		req.UserID = pid
	}

	key, questions, err := h.mgr.Questions(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := h.quiz.StartSessionWithQuestions(r.Context(), req.UserID, questions)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.mgr.Bind(result.SessionID, period, key)

	response := StartSessionResponse{
		SessionID:      result.SessionID,
		TotalQuestions: result.TotalQuestions,
		Question:       questionToDTO(result.FirstQuestion),
	}
	response.Question.MediaURL = protectedImageURL(result.SessionID, result.FirstQuestion)
	writeSuccess(w, map[string]interface{}{"key": key, "start": response})
}

func (h *ChallengeHandler) handleFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	period, key, ok := h.mgr.Lookup(req.SessionID)
	if !ok {
		writeError(w, http.StatusNotFound, "not a challenge session")
		return
	}
	session, err := h.quiz.GetSession(r.Context(), req.SessionID)
	if err != nil {
		writeSessionError(w, err)
		return
	}
	if h.auth != nil {
		pid, err := h.auth.Authenticate(bearerToken(r))
		if err != nil || pid == "" {
			writeError(w, http.StatusUnauthorized, "authentification requise")
			return
		}
		if session.UserID() != pid {
			writeError(w, http.StatusForbidden, "session d'un autre joueur")
			return
		}
	}

	// Score is read server-side from the session, not trusted from the client.
	username := session.UserID()
	if p, err := h.players.GetByID(r.Context(), session.UserID()); err == nil {
		username = p.Username()
	}
	if err := h.scores.SaveScore(r.Context(), string(period), key, session.UserID(), username,
		session.TotalScore(), session.CorrectCount(), session.QuestionsCount(),
		time.Now().UTC().Format(time.RFC3339)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	board, _ := h.scores.Leaderboard(r.Context(), string(period), key, 20)
	writeSuccess(w, map[string]interface{}{
		"score":       session.TotalScore(),
		"correct":     session.CorrectCount(),
		"total":       session.QuestionsCount(),
		"leaderboard": board,
	})
}
