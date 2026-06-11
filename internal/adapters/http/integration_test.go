package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
)

// wiredHandler builds a handler backed by a real quiz service over an in-memory
// database and the mock species source, for full-flow (success-path) tests.
func wiredHandler(t *testing.T) (*httphandler.Handler, string) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	playerRepo := sqlite.NewPlayerRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)
	factory := appquiz.NewQuestionFactory(mock.NewSpeciesRepository())
	svc := appquiz.NewService(factory, sessionRepo, playerRepo, nil)

	player, err := gamification.NewPlayer("u1", "Tester")
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	if err := playerRepo.Create(context.Background(), player); err != nil {
		t.Fatalf("create player: %v", err)
	}
	return httphandler.NewHandler(svc, false), "u1"
}

func postJSON(t *testing.T, h http.HandlerFunc, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h(rec, req)
	var resp struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
		Error   string         `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if !resp.Success {
		t.Fatalf("%s failed: status=%d err=%q", path, rec.Code, resp.Error)
	}
	return rec, resp.Data
}

// TestQuizFlow_StartAnswerComplete exercises the full happy path: start a
// session, answer every question, and reach completion.
func TestQuizFlow_StartAnswerComplete(t *testing.T) {
	handler, userID := wiredHandler(t)

	_, data := postJSON(t, handler.HandleStartSession, "/api/v1/quiz/start", map[string]any{
		"user_id":        userID,
		"difficulty":     "beginner",
		"quiz_types":     []string{"image"},
		"question_count": 3,
	})
	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		t.Fatal("no session_id returned")
	}
	question, _ := data["question"].(map[string]any)
	if question == nil {
		t.Fatal("no first question returned")
	}

	firstChoiceID := func(q map[string]any) int {
		choices, _ := q["choices"].([]any)
		if len(choices) == 0 {
			t.Fatal("question has no choices")
		}
		c, _ := choices[0].(map[string]any)
		id, _ := c["species_id"].(float64)
		return int(id)
	}

	complete := false
	for i := 0; i < 5 && !complete; i++ {
		_, ans := postJSON(t, handler.HandleSubmitAnswer, "/api/v1/quiz/answer", map[string]any{
			"session_id":    sessionID,
			"species_id":    firstChoiceID(question),
			"time_taken_ms": 1000,
		})
		complete, _ = ans["session_complete"].(bool)
		if next, ok := ans["next_question"].(map[string]any); ok {
			question = next
		}
	}
	if !complete {
		t.Error("session never completed after answering every question")
	}
}

// TestConfigAndPlayer covers two more success paths: /config and /players/{id}.
func TestConfigAndPlayer(t *testing.T) {
	handler, userID := wiredHandler(t)

	rec := httptest.NewRecorder()
	handler.HandleConfig(rec, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("HandleConfig status = %d, want 200", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/players/"+userID, nil)
	req.SetPathValue("id", userID)
	rec = httptest.NewRecorder()
	handler.HandleGetPlayer(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("HandleGetPlayer status = %d, want 200", rec.Code)
	}
}
