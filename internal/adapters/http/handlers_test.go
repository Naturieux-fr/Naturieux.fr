package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
)

// newTestHandler builds a handler backed by an empty service: no factory and
// no repositories, so session lookups return not-found.
func newTestHandler() *httphandler.Handler {
	return httphandler.NewHandler(appquiz.NewService(nil, nil, nil, nil), false)
}

func TestHandler_HandleHealthCheck(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleHealthCheck() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response httphandler.Response
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("HandleHealthCheck() Success = false, want true")
	}
}

func TestHandler_HandleHealthCheck_WrongMethod(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HandleHealthCheck(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleHealthCheck() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleStartSession_WrongMethod(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quiz/start", nil)
	rec := httptest.NewRecorder()

	handler.HandleStartSession(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleStartSession() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleStartSession_InvalidJSON(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/start", body)
	rec := httptest.NewRecorder()

	handler.HandleStartSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleStartSession() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleStartSession_MissingUserID(t *testing.T) {
	handler := newTestHandler()

	reqBody := httphandler.StartSessionRequest{
		Difficulty:    "beginner",
		QuestionCount: 5,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/start", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleStartSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleStartSession() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleSubmitAnswer_WrongMethod(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quiz/answer", nil)
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleSubmitAnswer_SessionNotFound(t *testing.T) {
	handler := newTestHandler()

	reqBody := httphandler.SubmitAnswerRequest{
		SessionID:   "nonexistent",
		SpeciesID:   1,
		TimeTakenMs: 5000,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/answer", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_HandleAbandonSession_WrongMethod(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quiz/abandon", nil)
	rec := httptest.NewRecorder()

	handler.HandleAbandonSession(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleAbandonSession() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleAbandonSession_SessionNotFound(t *testing.T) {
	handler := newTestHandler()

	reqBody := map[string]string{"session_id": "nonexistent"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/abandon", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleAbandonSession(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("HandleAbandonSession() status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	handler := newTestHandler()
	mux := http.NewServeMux()

	handler.RegisterRoutes(mux)

	// Test that routes are registered by checking health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("RegisterRoutes() health check status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_HandleSubmitAnswer_MissingSessionID(t *testing.T) {
	handler := newTestHandler()

	reqBody := httphandler.SubmitAnswerRequest{
		SpeciesID:   1,
		TimeTakenMs: 5000,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/answer", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleAbandonSession_InvalidJSON(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/abandon", body)
	rec := httptest.NewRecorder()

	handler.HandleAbandonSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleAbandonSession() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_HandleSubmitAnswer_InvalidJSON(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/answer", body)
	rec := httptest.NewRecorder()

	handler.HandleSubmitAnswer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleSubmitAnswer() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// newPlayerTestHandler builds a handler with a real SQLite player repository.
func newPlayerTestHandler(t *testing.T) *httphandler.Handler {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	service := appquiz.NewService(nil, nil, sqlite.NewPlayerRepository(db), nil)
	return httphandler.NewHandler(service, false)
}

func decodePlayer(t *testing.T, rec *httptest.ResponseRecorder) httphandler.PlayerDTO {
	t.Helper()
	var response struct {
		Data httphandler.PlayerDTO `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	return response.Data
}

func TestHandler_HandleRegisterPlayer(t *testing.T) {
	handler := newPlayerTestHandler(t)

	body, _ := json.Marshal(map[string]string{"username": "alice"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/players", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleRegisterPlayer(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleRegisterPlayer() status = %d, want %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	player := decodePlayer(t, rec)
	if player.ID == "" {
		t.Error("HandleRegisterPlayer() returned empty player id")
	}
	if player.Username != "alice" {
		t.Errorf("Username = %s, want alice", player.Username)
	}
	if player.Level != 1 {
		t.Errorf("Level = %d, want 1", player.Level)
	}
}

func TestHandler_HandleRegisterPlayer_InvalidUsername(t *testing.T) {
	handler := newPlayerTestHandler(t)

	for _, username := range []string{"", "a", "   ", "ce pseudo est beaucoup beaucoup trop long"} {
		body, _ := json.Marshal(map[string]string{"username": username})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/players", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.HandleRegisterPlayer(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("HandleRegisterPlayer(%q) status = %d, want %d", username, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestHandler_HandleRegisterPlayer_DuplicateUsername(t *testing.T) {
	handler := newPlayerTestHandler(t)

	body, _ := json.Marshal(map[string]string{"username": "alice"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/players", bytes.NewReader(body))
	handler.HandleRegisterPlayer(httptest.NewRecorder(), req)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/players", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.HandleRegisterPlayer(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("HandleRegisterPlayer(duplicate) status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestHandler_HandleGetPlayer(t *testing.T) {
	handler := newPlayerTestHandler(t)

	body, _ := json.Marshal(map[string]string{"username": "alice"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/players", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.HandleRegisterPlayer(rec, req)
	created := decodePlayer(t, rec)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/players/"+created.ID, nil)
	getReq.SetPathValue("id", created.ID)
	getRec := httptest.NewRecorder()

	handler.HandleGetPlayer(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("HandleGetPlayer() status = %d, want %d", getRec.Code, http.StatusOK)
	}
	fetched := decodePlayer(t, getRec)
	if fetched.ID != created.ID || fetched.Username != "alice" {
		t.Errorf("GetPlayer = %s/%s, want %s/alice", fetched.ID, fetched.Username, created.ID)
	}
}

func TestHandler_HandleGetPlayer_NotFound(t *testing.T) {
	handler := newPlayerTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/ghost", nil)
	req.SetPathValue("id", "ghost")
	rec := httptest.NewRecorder()

	handler.HandleGetPlayer(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("HandleGetPlayer() status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_HandleLeaderboard_WrongMethod(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/leaderboard", nil)
	rec := httptest.NewRecorder()

	handler.HandleLeaderboard(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleLeaderboard() status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandler_HandleLeaderboard_InvalidLimit(t *testing.T) {
	handler := newTestHandler()

	for _, limit := range []string{"abc", "-3", "0"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/leaderboard?limit="+limit, nil)
		rec := httptest.NewRecorder()

		handler.HandleLeaderboard(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("HandleLeaderboard(limit=%s) status = %d, want %d", limit, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestHandler_HandleLeaderboard_ReturnsRankedPlayers(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	playerRepo := sqlite.NewPlayerRepository(db)
	ctx := context.Background()
	for _, p := range []struct {
		id, name string
		xp       int
	}{
		{"p1", "alice", 100},
		{"p2", "bob", 500},
	} {
		player, _ := gamification.NewPlayer(p.id, p.name)
		player.AddXP(p.xp)
		if err := playerRepo.Create(ctx, player); err != nil {
			t.Fatalf("Create(%s) error = %v", p.id, err)
		}
	}

	service := appquiz.NewService(nil, nil, playerRepo, nil)
	handler := httphandler.NewHandler(service, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/leaderboard?limit=10", nil)
	rec := httptest.NewRecorder()

	handler.HandleLeaderboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HandleLeaderboard() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Entries []httphandler.LeaderboardEntryDTO `json:"entries"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	entries := response.Data.Entries
	if len(entries) != 2 {
		t.Fatalf("Entries = %d, want 2", len(entries))
	}
	if entries[0].Username != "bob" || entries[0].Rank != 1 {
		t.Errorf("First entry = %s (rank %d), want bob (rank 1)", entries[0].Username, entries[0].Rank)
	}
	if entries[1].Username != "alice" || entries[1].Rank != 2 {
		t.Errorf("Second entry = %s (rank %d), want alice (rank 2)", entries[1].Username, entries[1].Rank)
	}
}
