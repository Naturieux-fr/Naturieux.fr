// Package http provides HTTP handlers for the API.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

// Handler contains all HTTP handlers.
type Handler struct {
	quizService *appquiz.Service
	sessions    map[string]*quiz.Session // In-memory session store (use proper storage in production)
}

// NewHandler creates a new Handler.
func NewHandler(quizService *appquiz.Service) *Handler {
	return &Handler{
		quizService: quizService,
		sessions:    make(map[string]*quiz.Session),
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
	ID            string      `json:"id"`
	QuizType      string      `json:"quiz_type"`
	Difficulty    string      `json:"difficulty"`
	MediaURL      string      `json:"media_url"`
	TimeLimit     int         `json:"time_limit_seconds"`
	FlashDuration int         `json:"flash_duration_ms,omitempty"`
	Choices       []ChoiceDTO `json:"choices"`
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

	// Store session (in production, use proper storage)
	// Note: This is simplified for the example
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

	// Get session (in production, use proper storage)
	session, ok := h.sessions[req.SessionID]
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
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

	session, ok := h.sessions[req.SessionID]
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	if err := h.quizService.AbandonSession(r.Context(), session); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	delete(h.sessions, req.SessionID)
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
		ID:         q.ID(),
		QuizType:   string(q.QuizType()),
		Difficulty: string(q.Difficulty()),
		MediaURL:   q.MediaURL(),
		TimeLimit:  int(q.TimeLimit().Seconds()),
		Choices:    choices,
	}

	if q.QuizType() == quiz.FlashQuiz {
		dto.FlashDuration = int(q.FlashDuration().Milliseconds())
	}

	return dto
}

// RegisterRoutes registers all routes with the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.HandleHealthCheck)
	mux.HandleFunc("/api/v1/quiz/start", h.HandleStartSession)
	mux.HandleFunc("/api/v1/quiz/answer", h.HandleSubmitAnswer)
	mux.HandleFunc("/api/v1/quiz/abandon", h.HandleAbandonSession)
}
