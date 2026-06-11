// Package http provides HTTP handlers for the API.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// Handler contains all HTTP handlers.
type Handler struct {
	quizService      *appquiz.Service
	devMode          bool
	mediaDir         string // local media directory, for serving owned photos
	registrationMode string // "open" or "invite" (surfaced in /config)
	auth             Authenticator
}

// SetRegistrationMode records the account registration policy for /config.
func (h *Handler) SetRegistrationMode(mode string) {
	h.registrationMode = mode
}

// NewHandler creates a new Handler.
func NewHandler(quizService *appquiz.Service, devMode bool) *Handler {
	return &Handler{
		quizService: quizService,
		devMode:     devMode,
	}
}

// SetLocalMediaDir lets the quiz-image proxy read owned photos from disk.
func (h *Handler) SetLocalMediaDir(dir string) {
	h.mediaDir = dir
}

// Authenticator resolves a session token to a player id.
type Authenticator interface {
	Authenticate(token string) (string, error)
}

// SetAuthenticator enables player-token auth on gameplay endpoints. When set,
// the player is derived from the verified token instead of trusting a
// client-supplied user_id, so scores and leaderboards can't be forged.
func (h *Handler) SetAuthenticator(a Authenticator) { h.auth = a }

// resolveUserID returns the authenticated player id (ignoring the body value)
// when auth is enabled, or the body value otherwise. It writes a 401 and
// returns ok=false when a required token is missing or invalid.
func (h *Handler) resolveUserID(w http.ResponseWriter, r *http.Request, bodyID string) (string, bool) {
	if h.auth == nil {
		return bodyID, true
	}
	pid, err := h.auth.Authenticate(bearerToken(r))
	if err != nil || pid == "" {
		writeError(w, http.StatusUnauthorized, "authentification requise")
		return "", false
	}
	return pid, true
}

// ownsSession verifies the caller owns the session (when auth is enabled).
func (h *Handler) ownsSession(w http.ResponseWriter, r *http.Request, session *quiz.Session) bool {
	if h.auth == nil {
		return true
	}
	pid, err := h.auth.Authenticate(bearerToken(r))
	if err != nil || pid == "" {
		writeError(w, http.StatusUnauthorized, "authentification requise")
		return false
	}
	if session.UserID() != pid {
		writeError(w, http.StatusForbidden, "session d'un autre joueur")
		return false
	}
	return true
}

// Response represents a standard API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data) // Error ignored: response already sent
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{
		Success: false,
		Error:   message,
	})
}

// writeSessionError maps a session lookup error to an HTTP response.
func writeSessionError(w http.ResponseWriter, err error) {
	if errors.Is(err, ports.ErrNotFound) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

// writeSuccess writes a success response.
func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// StartSessionRequest represents a request to start a new quiz session.
type StartSessionRequest struct {
	UserID        string   `json:"user_id"`
	Difficulty    string   `json:"difficulty"`
	QuizTypes     []string `json:"quiz_types"`
	TaxonFilter   string   `json:"taxon_filter"`
	QuestionCount int      `json:"question_count"`
}

// StartSessionResponse represents the response for starting a session.
type StartSessionResponse struct {
	SessionID      string      `json:"session_id"`
	TotalQuestions int         `json:"total_questions"`
	Question       QuestionDTO `json:"question"`
}

// QuestionDTO represents a question for API responses.
type QuestionDTO struct {
	ID               string      `json:"id"`
	QuizType         string      `json:"quiz_type"`
	Difficulty       string      `json:"difficulty"`
	MediaURL         string      `json:"media_url"`
	MediaAttribution string      `json:"media_attribution,omitempty"`
	MediaLicense     string      `json:"media_license,omitempty"`
	TimeLimit        int         `json:"time_limit_seconds"`
	FlashDuration    int         `json:"flash_duration_ms,omitempty"`
	Choices          []ChoiceDTO `json:"choices"`
}

// ChoiceDTO represents a choice for API responses.
type ChoiceDTO struct {
	SpeciesID   int    `json:"species_id"`
	DisplayName string `json:"display_name"`
}

// SubmitAnswerRequest represents a request to submit an answer.
type SubmitAnswerRequest struct {
	SessionID   string `json:"session_id"`
	SpeciesID   int    `json:"species_id"`
	TimeTakenMs int    `json:"time_taken_ms"`
}

// SubmitAnswerResponse represents the response for submitting an answer.
type SubmitAnswerResponse struct {
	IsCorrect        bool         `json:"is_correct"`
	Score            int          `json:"score"`
	CorrectSpeciesID int          `json:"correct_species_id"`
	CorrectName      string       `json:"correct_name"`
	CurrentStreak    int          `json:"current_streak"`
	TotalScore       int          `json:"total_score"`
	Accuracy         float64      `json:"accuracy"`
	SessionComplete  bool         `json:"session_complete"`
	NextQuestion     *QuestionDTO `json:"next_question,omitempty"`
}

// HandleStartSession handles POST /api/v1/quiz/start
func (h *Handler) HandleStartSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, ok := h.resolveUserID(w, r, req.UserID)
	if !ok {
		return
	}
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	req.UserID = userID

	// Convert quiz types
	quizTypes := make([]quiz.QuizType, 0, len(req.QuizTypes))
	for _, qt := range req.QuizTypes {
		quizTypes = append(quizTypes, quiz.QuizType(qt))
	}

	serviceReq := appquiz.StartSessionRequest{
		UserID:        req.UserID,
		Difficulty:    quiz.Difficulty(req.Difficulty),
		QuizTypes:     quizTypes,
		TaxonFilter:   req.TaxonFilter,
		QuestionCount: req.QuestionCount,
	}

	result, err := h.quizService.StartSession(r.Context(), serviceReq)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := StartSessionResponse{
		SessionID:      result.SessionID,
		TotalQuestions: result.TotalQuestions,
		Question:       questionToDTO(result.FirstQuestion),
	}
	// Serve the image through an opaque proxy so the real file/species URL
	// never reaches the client.
	response.Question.MediaURL = protectedImageURL(result.SessionID, result.FirstQuestion)

	writeSuccess(w, response)
}

// HandleSubmitAnswer handles POST /api/v1/quiz/answer
func (h *Handler) HandleSubmitAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SubmitAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	session, err := h.quizService.GetSession(r.Context(), req.SessionID)
	if err != nil {
		writeSessionError(w, err)
		return
	}
	if !h.ownsSession(w, r, session) {
		return
	}

	serviceReq := appquiz.SubmitAnswerRequest{
		SessionID: req.SessionID,
		SpeciesID: req.SpeciesID,
		TimeTaken: time.Duration(req.TimeTakenMs) * time.Millisecond,
	}

	result, err := h.quizService.SubmitAnswer(r.Context(), session, serviceReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := SubmitAnswerResponse{
		IsCorrect:        result.IsCorrect,
		Score:            result.Score,
		CorrectSpeciesID: result.CorrectSpeciesID,
		CorrectName:      result.CorrectName,
		CurrentStreak:    result.CurrentStreak,
		TotalScore:       result.TotalScore,
		Accuracy:         result.Accuracy,
		SessionComplete:  result.SessionComplete,
	}

	if result.NextQuestion != nil {
		dto := questionToDTO(result.NextQuestion)
		dto.MediaURL = protectedImageURL(req.SessionID, result.NextQuestion)
		response.NextQuestion = &dto
	}

	writeSuccess(w, response)
}

// HandleAbandonSession handles POST /api/v1/quiz/abandon
func (h *Handler) HandleAbandonSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.quizService.GetSession(r.Context(), req.SessionID)
	if err != nil {
		writeSessionError(w, err)
		return
	}

	if err := h.quizService.AbandonSession(r.Context(), session); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, map[string]string{"message": "session abandoned"})
}

// HandleHealthCheck handles GET /health
func (h *Handler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeSuccess(w, map[string]string{
		"status":  "healthy",
		"service": "naturieux-api",
	})
}

// questionToDTO converts a domain Question to a DTO.
func questionToDTO(q *quiz.Question) QuestionDTO {
	choices := make([]ChoiceDTO, len(q.Choices()))
	for i, c := range q.Choices() {
		choices[i] = ChoiceDTO{
			SpeciesID:   c.Species.ID(),
			DisplayName: c.Species.DisplayName(),
		}
	}

	dto := QuestionDTO{
		ID:               q.ID(),
		QuizType:         string(q.QuizType()),
		Difficulty:       string(q.Difficulty()),
		MediaURL:         q.MediaURL(),
		MediaAttribution: q.MediaAttribution(),
		MediaLicense:     q.MediaLicense(),
		TimeLimit:        int(q.TimeLimit().Seconds()),
		Choices:          choices,
	}

	if q.QuizType() == quiz.FlashQuiz {
		dto.FlashDuration = int(q.FlashDuration().Milliseconds())
	}

	return dto
}

// PlayerDTO represents a player profile for API responses.
type PlayerDTO struct {
	ID              string         `json:"id"`
	Username        string         `json:"username"`
	Level           int            `json:"level"`
	TotalXP         int            `json:"total_xp"`
	XPInLevel       int            `json:"xp_in_level"`
	XPForLevel      int            `json:"xp_for_level"`
	TotalGames      int            `json:"total_games"`
	Accuracy        float64        `json:"accuracy"`
	BestStreak      int            `json:"best_streak"`
	DailyStreak     int            `json:"daily_streak"`
	Achievements    []string       `json:"achievements"`
	CategoryCorrect map[string]int `json:"category_correct"`
}

// playerToDTO converts a domain Player to a DTO.
func playerToDTO(p *gamification.Player) PlayerDTO {
	achievements := make([]string, len(p.Achievements()))
	for i, a := range p.Achievements() {
		achievements[i] = string(a)
	}

	xpForLevel := gamification.XPForLevel(p.Level())
	return PlayerDTO{
		ID:              p.ID(),
		Username:        p.Username(),
		Level:           p.Level(),
		TotalXP:         p.TotalXP(),
		XPInLevel:       xpForLevel - p.XPToNextLevel(),
		XPForLevel:      xpForLevel,
		TotalGames:      p.TotalGames(),
		Accuracy:        p.Accuracy(),
		BestStreak:      p.BestStreak(),
		DailyStreak:     p.DailyStreak(),
		Achievements:    achievements,
		CategoryCorrect: p.CategoryCorrect(),
	}
}

// HandleRegisterPlayer handles POST /api/v1/players (legacy passwordless
// registration). Disabled when registration is invite-only so it cannot be
// used to bypass invitations; real accounts go through /api/v1/account.
func (h *Handler) HandleRegisterPlayer(w http.ResponseWriter, r *http.Request) {
	if h.registrationMode == "invite" {
		writeError(w, http.StatusForbidden, "inscription sur invitation uniquement")
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	player, err := h.quizService.RegisterPlayer(r.Context(), req.Username)
	if err != nil {
		switch {
		case errors.Is(err, appquiz.ErrInvalidUsername):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ports.ErrAlreadyExists):
			writeError(w, http.StatusConflict, "username already taken")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeSuccess(w, playerToDTO(player))
}

// HandleGetPlayer handles GET /api/v1/players/{id}
func (h *Handler) HandleGetPlayer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "player id is required")
		return
	}

	player, err := h.quizService.GetPlayer(r.Context(), id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeSuccess(w, playerToDTO(player))
}

// LeaderboardEntryDTO represents one ranked player for API responses.
type LeaderboardEntryDTO struct {
	Rank       int     `json:"rank"`
	Username   string  `json:"username"`
	Level      int     `json:"level"`
	TotalXP    int     `json:"total_xp"`
	TotalGames int     `json:"total_games"`
	Accuracy   float64 `json:"accuracy"`
	BestStreak int     `json:"best_streak"`
}

// Leaderboard size limits.
const (
	defaultLeaderboardLimit = 10
	maxLeaderboardLimit     = 100
)

// HandleLeaderboard handles GET /api/v1/leaderboard
func (h *Handler) HandleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := defaultLeaderboardLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = min(parsed, maxLeaderboardLimit)
	}

	players, err := h.quizService.GetLeaderboard(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	entries := make([]LeaderboardEntryDTO, len(players))
	for i, p := range players {
		entries[i] = LeaderboardEntryDTO{
			Rank:       i + 1,
			Username:   p.Username(),
			Level:      p.Level(),
			TotalXP:    p.TotalXP(),
			TotalGames: p.TotalGames(),
			Accuracy:   p.Accuracy(),
			BestStreak: p.BestStreak(),
		}
	}

	writeSuccess(w, map[string]interface{}{"entries": entries})
}

// HandleConfig handles GET /api/v1/config
func (h *Handler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	mode := h.registrationMode
	if mode == "" {
		mode = "open"
	}
	writeSuccess(w, map[string]interface{}{
		"dev_mode":          h.devMode,
		"registration_mode": mode,
		"version":           "1.0.0",
	})
}

// protectedImageURL returns the opaque proxy URL for a question's image, or
// "" when the question has no media. The question id is appended so the
// browser never reuses a cached response across questions.
func protectedImageURL(sessionID string, q *quiz.Question) string {
	if q == nil || q.MediaURL() == "" {
		return ""
	}
	return "/api/v1/quiz/" + sessionID + "/image?n=" + url.QueryEscape(q.ID())
}

// HandleGuess checks a free-text guess against the current question without
// advancing it. A wrong guess reveals nothing; a correct guess returns the
// species id so the client can record the answer through the normal flow.
func (h *Handler) HandleGuess(w http.ResponseWriter, r *http.Request) {
	session, err := h.quizService.GetSession(r.Context(), r.PathValue("session_id"))
	if err != nil {
		writeSessionError(w, err)
		return
	}
	if !h.ownsSession(w, r, session) {
		return
	}
	q := session.CurrentQuestion()
	if q == nil {
		writeError(w, http.StatusConflict, "no current question")
		return
	}

	var req struct {
		Guess string `json:"guess"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if q.CorrectSpecies().MatchesName(req.Guess) {
		writeSuccess(w, map[string]interface{}{"correct": true, "species_id": q.CorrectSpecies().ID()})
		return
	}
	writeSuccess(w, map[string]interface{}{"correct": false})
}

// HandleQuizImage streams the current question's image without exposing the
// underlying file or species URL. It only serves while the session is in
// progress, and only the current question, so the response cannot be replayed
// to enumerate the collection.
func (h *Handler) HandleQuizImage(w http.ResponseWriter, r *http.Request) {
	session, err := h.quizService.GetSession(r.Context(), r.PathValue("session_id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if session.Status() != quiz.SessionInProgress {
		http.NotFound(w, r)
		return
	}
	q := session.CurrentQuestion()
	if q == nil || q.MediaURL() == "" {
		http.NotFound(w, r)
		return
	}
	if n := r.URL.Query().Get("n"); n != "" && n != q.ID() {
		http.NotFound(w, r)
		return
	}
	h.streamImage(w, r, q.MediaURL())
}

// streamImage proxies an image (a local owned photo or a remote CC URL) with
// caching disabled, so the bytes are not retained by the browser.
func (h *Handler) streamImage(w http.ResponseWriter, r *http.Request, mediaURL string) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")

	if strings.HasPrefix(mediaURL, "/media/") {
		if h.mediaDir == "" {
			http.NotFound(w, r)
			return
		}
		name := path.Base(mediaURL)
		if name == "." || name == "/" || strings.Contains(name, "..") {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		http.ServeFile(w, r, filepath.Join(h.mediaDir, name))
		return
	}

	// Remote (e.g. iNaturalist / S3): proxy the bytes server-side.
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, mediaURL, nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	req.Header.Set("User-Agent", "Naturieux/1.0 (https://naturieux.fr)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "image unavailable", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "image unavailable", http.StatusBadGateway)
		return
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, _ = io.Copy(w, resp.Body)
}

// RegisterRoutes registers all routes with the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.HandleHealthCheck)
	mux.HandleFunc("/api/v1/config", h.HandleConfig)
	mux.HandleFunc("/api/v1/leaderboard", h.HandleLeaderboard)
	mux.HandleFunc("POST /api/v1/players", h.HandleRegisterPlayer)
	mux.HandleFunc("GET /api/v1/players/{id}", h.HandleGetPlayer)
	mux.HandleFunc("/api/v1/quiz/start", h.HandleStartSession)
	mux.HandleFunc("/api/v1/quiz/answer", h.HandleSubmitAnswer)
	mux.HandleFunc("/api/v1/quiz/abandon", h.HandleAbandonSession)
	mux.HandleFunc("GET /api/v1/quiz/{session_id}/image", h.HandleQuizImage)
	mux.HandleFunc("POST /api/v1/quiz/{session_id}/guess", h.HandleGuess)
}
