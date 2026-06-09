// Package http provides HTTP handlers for the API.
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// Handler contains all HTTP handlers.
type Handler struct {
	quizService *appquiz.Service
	devMode     bool
}

// NewHandler creates a new Handler.
func NewHandler(quizService *appquiz.Service, devMode bool) *Handler {
	return &Handler{
		quizService: quizService,
		devMode:     devMode,
	}
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

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

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
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := StartSessionResponse{
		SessionID:      result.SessionID,
		TotalQuestions: result.TotalQuestions,
		Question:       questionToDTO(result.FirstQuestion),
	}

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

	writeSuccess(w, map[string]interface{}{
		"dev_mode": h.devMode,
		"version":  "1.0.0",
	})
}

// RegisterRoutes registers all routes with the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.HandleHealthCheck)
	mux.HandleFunc("/api/v1/config", h.HandleConfig)
	mux.HandleFunc("/api/v1/leaderboard", h.HandleLeaderboard)
	mux.HandleFunc("/api/v1/quiz/start", h.HandleStartSession)
	mux.HandleFunc("/api/v1/quiz/answer", h.HandleSubmitAnswer)
	mux.HandleFunc("/api/v1/quiz/abandon", h.HandleAbandonSession)
}
