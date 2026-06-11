package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

// RoomHandler serves the multiplayer room API. It reuses the main handler's
// image proxy so room photos are protected exactly like solo ones.
type RoomHandler struct {
	rooms  *room.Manager
	images *Handler
}

// NewRoomHandler creates the multiplayer API handler.
func NewRoomHandler(rooms *room.Manager, images *Handler) *RoomHandler {
	return &RoomHandler{rooms: rooms, images: images}
}

// RegisterRoutes wires the room endpoints.
func (h *RoomHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/rooms", h.handleCreate)
	mux.HandleFunc("POST /api/v1/rooms/{code}/join", h.handleJoin)
	mux.HandleFunc("POST /api/v1/rooms/{code}/start", h.handleStart)
	mux.HandleFunc("GET /api/v1/rooms/{code}", h.handleState)
	mux.HandleFunc("POST /api/v1/rooms/{code}/answer", h.handleAnswer)
	mux.HandleFunc("POST /api/v1/rooms/{code}/guess", h.handleGuess)
	mux.HandleFunc("GET /api/v1/rooms/{code}/image", h.handleImage)
}

func (h *RoomHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HostID      string `json:"host_id"`
		HostName    string `json:"host_name"`
		Difficulty  string `json:"difficulty"`
		QuizType    string `json:"quiz_type"`
		TaxonFilter string `json:"taxon_filter"`
		Count       int    `json:"count"`
		Mode        string `json:"mode"`
		AnswerMode  string `json:"answer_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rm, token, err := h.rooms.Create(req.HostID, req.HostName, room.Settings{
		Difficulty:  quiz.Difficulty(req.Difficulty),
		QuizType:    quiz.QuizType(orDefault(req.QuizType, "image")),
		TaxonFilter: req.TaxonFilter,
		Count:       req.Count,
		Mode:        room.Mode(orDefault(req.Mode, "classic")),
		AnswerMode:  orDefault(req.AnswerMode, "choices"),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"code": rm.Code(), "token": token})
}

func (h *RoomHandler) handleJoin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID   string `json:"player_id"`
		PlayerName string `json:"player_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	_, token, err := h.rooms.Join(r.PathValue("code"), req.PlayerID, req.PlayerName)
	if err != nil {
		h.writeRoomError(w, err)
		return
	}
	h.respondStateWith(w, r.PathValue("code"), req.PlayerID, map[string]interface{}{"token": token})
}

func (h *RoomHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"player_id"`
		Token    string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.rooms.Start(r.Context(), r.PathValue("code"), req.Token); err != nil {
		h.writeRoomError(w, err)
		return
	}
	h.respondState(w, r.PathValue("code"), req.PlayerID)
}

func (h *RoomHandler) handleState(w http.ResponseWriter, r *http.Request) {
	h.respondState(w, r.PathValue("code"), r.URL.Query().Get("player_id"))
}

func (h *RoomHandler) handleAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID  string `json:"player_id"`
		Token     string `json:"token"`
		SpeciesID int    `json:"species_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	code := r.PathValue("code")
	res, err := h.rooms.Answer(code, req.Token, req.SpeciesID)
	if err != nil {
		h.writeRoomError(w, err)
		return
	}
	out := map[string]interface{}{
		"is_correct":         res.IsCorrect,
		"correct_name":       res.CorrectName,
		"correct_species_id": res.CorrectSpeciesID,
		"score":              res.Score,
		"total_score":        res.TotalScore,
		"streak":             res.Streak,
		"done":               res.Done,
		"eliminated":         res.Eliminated,
	}
	if res.NextQuestion != nil {
		dto := questionToDTO(res.NextQuestion)
		dto.MediaURL = h.roomImageURL(code, req.PlayerID, res.NextQuestion)
		out["next_question"] = dto
	}
	writeSuccess(w, out)
}

func (h *RoomHandler) handleGuess(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
		Guess string `json:"guess"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	correct, speciesID, err := h.rooms.GuessName(r.PathValue("code"), req.Token, req.Guess)
	if err != nil {
		h.writeRoomError(w, err)
		return
	}
	if correct {
		writeSuccess(w, map[string]interface{}{"correct": true, "species_id": speciesID})
		return
	}
	writeSuccess(w, map[string]interface{}{"correct": false})
}

func (h *RoomHandler) handleImage(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	playerID := r.URL.Query().Get("player_id")
	q, err := h.rooms.CurrentQuestion(code, playerID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if n := r.URL.Query().Get("n"); n != "" && n != q.ID() {
		http.NotFound(w, r)
		return
	}
	h.images.streamImage(w, r, q.MediaURL())
}

// respondState writes the room snapshot, plus the viewer's current question
// when the game is running and they still have one to answer.
func (h *RoomHandler) respondState(w http.ResponseWriter, code, viewerID string) {
	h.respondStateWith(w, code, viewerID, nil)
}

// respondStateWith is respondState with extra top-level fields merged in.
func (h *RoomHandler) respondStateWith(w http.ResponseWriter, code, viewerID string, extra map[string]interface{}) {
	state, err := h.rooms.Snapshot(code)
	if err != nil {
		h.writeRoomError(w, err)
		return
	}
	payload := map[string]interface{}{"room": state}
	for k, v := range extra {
		payload[k] = v
	}

	if state.Status == room.Playing && viewerID != "" {
		if q, err := h.rooms.CurrentQuestion(code, viewerID); err == nil {
			dto := questionToDTO(q)
			dto.MediaURL = h.roomImageURL(code, viewerID, q)
			payload["your_question"] = dto
		}
	}
	writeSuccess(w, payload)
}

// roomImageURL builds the opaque proxy URL for a player's current question.
func (h *RoomHandler) roomImageURL(code, playerID string, q *quiz.Question) string {
	if q == nil || q.MediaURL() == "" {
		return ""
	}
	return "/api/v1/rooms/" + code + "/image?player_id=" + url.QueryEscape(playerID) + "&n=" + url.QueryEscape(q.ID())
}

// writeRoomError maps room errors to HTTP statuses.
func (h *RoomHandler) writeRoomError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, room.ErrNotFound), errors.Is(err, room.ErrPlayerUnknown):
		writeError(w, http.StatusNotFound, "room not found")
	case errors.Is(err, room.ErrForbidden):
		writeError(w, http.StatusForbidden, "only the host can do that")
	case errors.Is(err, room.ErrBadState):
		writeError(w, http.StatusConflict, "the room is not in the right state")
	case errors.Is(err, room.ErrNoQuestions):
		writeError(w, http.StatusInternalServerError, "could not prepare questions")
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

// orDefault returns v or a fallback when v is empty.
func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
