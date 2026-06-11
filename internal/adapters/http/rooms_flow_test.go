package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
)

func TestRoomAnswerFlow(t *testing.T) {
	mgr := room.NewManager(appquiz.NewQuestionFactory(mock.NewSpeciesRepository()))
	rh := httphandler.NewRoomHandler(mgr, httphandler.NewHandler(nil, false))
	mux := http.NewServeMux()
	rh.RegisterRoutes(mux)

	do := func(method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, jsonReq(method, path, body))
		_, d := decode(t, rec)
		return rec, d
	}

	_, c := do(http.MethodPost, "/api/v1/rooms", map[string]any{
		"host_id": "h1", "host_name": "Host", "count": 2, "difficulty": "beginner", "answer_mode": "free",
	})
	code := c["code"].(string)
	hostTok := c["token"].(string)
	_, j := do(http.MethodPost, "/api/v1/rooms/"+code+"/join", map[string]any{"player_id": "p2", "player_name": "Bob"})
	p2Tok := j["token"].(string)
	do(http.MethodPost, "/api/v1/rooms/"+code+"/start", map[string]any{"player_id": "h1", "token": hostTok})

	// Free-text guess for each player.
	if rec, _ := do(http.MethodPost, "/api/v1/rooms/"+code+"/guess", map[string]any{"token": hostTok, "guess": "nope"}); rec.Code != http.StatusOK {
		t.Errorf("host guess = %d", rec.Code)
	}
	if rec, _ := do(http.MethodPost, "/api/v1/rooms/"+code+"/answer", map[string]any{"token": p2Tok, "species_id": 1}); rec.Code != http.StatusOK {
		t.Errorf("p2 answer = %d", rec.Code)
	}
	// Bad token is rejected.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/rooms/"+code+"/answer", map[string]any{"token": "bad", "species_id": 1}))
	if rec.Code == http.StatusOK {
		t.Error("answer with bad token should fail")
	}
	// State for a spectator (no player_id).
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/"+code, nil))
	if rec.Code != http.StatusOK {
		t.Errorf("spectator state = %d", rec.Code)
	}
	// Unknown room.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/ZZZZ", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown room = %d", rec.Code)
	}
}
