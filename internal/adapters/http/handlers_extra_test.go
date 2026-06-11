package http_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
)

// --- helpers -------------------------------------------------------------

func memDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) (bool, map[string]any) {
	t.Helper()
	var resp struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.Success, resp.Data
}

func jsonReq(method, path string, body any) *http.Request {
	raw, _ := json.Marshal(body)
	return httptest.NewRequest(method, path, bytes.NewReader(raw))
}

// --- account: register, me, reset ---------------------------------------

func TestAccountHandlers(t *testing.T) {
	db := memDB(t)
	playerRepo := sqlite.NewPlayerRepository(db)
	svc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Open)
	svc.SetResetStore(sqlite.NewResetRepository(db))
	h := httphandler.NewAccountHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Register.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/account/register", map[string]any{"username": "Alice", "password": "secret1"}))
	ok, data := decode(t, rec)
	if !ok {
		t.Fatalf("register failed: %s", rec.Body)
	}
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatal("no token")
	}

	// /me with the token.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/account/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(rec, req)
	if ok, d := decode(t, rec); !ok || d["username"] != "Alice" {
		t.Errorf("me = %v / %s", ok, rec.Body)
	}

	// Wrong login.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/account/login", map[string]any{"username": "Alice", "password": "nope"}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad login status = %d", rec.Code)
	}

	// Admin issues a reset, the player sets a new password.
	pid := data["player"].(map[string]any)["id"].(string)
	resetTok, err := svc.IssueReset(context.Background(), pid)
	if err != nil {
		t.Fatalf("issue reset: %v", err)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/account/reset", map[string]any{"token": resetTok, "password": "brandnew1"}))
	if ok, _ := decode(t, rec); !ok {
		t.Errorf("reset failed: %s", rec.Body)
	}
}

// --- articles: writer authoring + public reading ------------------------

func TestArticleHandlers(t *testing.T) {
	db := memDB(t)
	playerRepo := sqlite.NewPlayerRepository(db)
	svc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Open)
	articleH := httphandler.NewArticleHandler(sqlite.NewArticleRepository(db), svc)
	accountH := httphandler.NewAccountHandler(svc)
	mux := http.NewServeMux()
	articleH.RegisterRoutes(mux)
	accountH.RegisterRoutes(mux)

	// Register a player and promote them to writer.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/account/register", map[string]any{"username": "Red", "password": "secret1"}))
	_, data := decode(t, rec)
	token := data["token"].(string)
	pid := data["player"].(map[string]any)["id"].(string)
	if err := playerRepo.SetRole(context.Background(), pid, "writer"); err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Create an article.
	rec = httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/api/v1/articles", map[string]any{
		"title": "Triton", "body": "comment distinguer…",
		"species": []map[string]any{{"cd_nom": 42, "name": "Triton"}}, "published": true,
	})
	req.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(rec, req)
	if ok, _ := decode(t, rec); !ok {
		t.Fatalf("create article: %s", rec.Body)
	}

	// Public listing and by-species.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/articles", nil))
	if ok, d := decode(t, rec); !ok || len(d["articles"].([]any)) != 1 {
		t.Errorf("list = %s", rec.Body)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/articles/by-species/42", nil))
	if ok, d := decode(t, rec); !ok || len(d["articles"].([]any)) != 1 {
		t.Errorf("by-species = %s", rec.Body)
	}

	// A non-writer cannot create.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/articles", map[string]any{"title": "x", "body": "y"}))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anon create status = %d", rec.Code)
	}
}

// --- challenge: state, start, finish ------------------------------------

func TestChallengeHandlers(t *testing.T) {
	db := memDB(t)
	playerRepo := sqlite.NewPlayerRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)
	factory := appquiz.NewQuestionFactory(mock.NewSpeciesRepository())
	quizSvc := appquiz.NewService(factory, sessionRepo, playerRepo, nil)
	mgr := challenge.NewManager(factory, sqlite.NewCuratedRepository(db))
	acc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Open)
	ch := httphandler.NewChallengeHandler(mgr, quizSvc, sqlite.NewChallengeRepository(db), playerRepo)
	ch.SetAuthenticator(acc)
	accH := httphandler.NewAccountHandler(acc)
	mux := http.NewServeMux()
	ch.RegisterRoutes(mux)
	accH.RegisterRoutes(mux)

	// State of an empty daily challenge.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/challenge/daily", nil))
	if ok, _ := decode(t, rec); !ok {
		t.Fatalf("challenge state: %s", rec.Body)
	}
	// Unknown period -> 404.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/challenge/yearly", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown period status = %d", rec.Code)
	}

	// Register, start the challenge, play to completion, finish.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/account/register", map[string]any{"username": "Carol", "password": "secret1"}))
	_, data := decode(t, rec)
	token := data["token"].(string)

	rec = httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/api/v1/challenge/daily/start", map[string]any{})
	req.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(rec, req)
	ok, start := decode(t, rec)
	if !ok {
		t.Fatalf("challenge start: %s", rec.Body)
	}
	sid := start["start"].(map[string]any)["session_id"].(string)

	// Finishing before completion is rejected.
	rec = httptest.NewRecorder()
	req = jsonReq(http.MethodPost, "/api/v1/challenge/finish", map[string]any{"session_id": sid})
	req.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("premature finish status = %d, want 409", rec.Code)
	}
}

// --- rooms: create, list, join, start, answer ---------------------------

func TestRoomHandlers(t *testing.T) {
	factory := appquiz.NewQuestionFactory(mock.NewSpeciesRepository())
	mgr := room.NewManager(factory)
	rh := httphandler.NewRoomHandler(mgr, httphandler.NewHandler(nil, false))
	mux := http.NewServeMux()
	rh.RegisterRoutes(mux)

	// Create a public room.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/rooms", map[string]any{
		"host_id": "h1", "host_name": "Host", "public": true, "count": 3, "difficulty": "beginner",
	}))
	ok, data := decode(t, rec)
	if !ok {
		t.Fatalf("create room: %s", rec.Body)
	}
	code := data["code"].(string)
	hostTok := data["token"].(string)

	// It shows up in the public list.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/rooms", nil))
	if ok, d := decode(t, rec); !ok || len(d["rooms"].([]any)) != 1 {
		t.Errorf("public rooms = %s", rec.Body)
	}

	// Another player joins.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/rooms/"+code+"/join", map[string]any{
		"player_id": "p2", "player_name": "Bob",
	}))
	if ok, _ := decode(t, rec); !ok {
		t.Fatalf("join: %s", rec.Body)
	}

	// Host starts the game.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/rooms/"+code+"/start", map[string]any{
		"player_id": "h1", "token": hostTok,
	}))
	if ok, _ := decode(t, rec); !ok {
		t.Fatalf("start: %s", rec.Body)
	}

	// State for the host includes their current question.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/"+code+"?player_id=h1", nil))
	if ok, _ := decode(t, rec); !ok {
		t.Errorf("state: %s", rec.Body)
	}
}

// --- locate: empty (no annotated photos) --------------------------------

func TestLocateHandler_Empty(t *testing.T) {
	db := memDB(t)
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("taxref schema: %v", err)
	}
	repo := taxref.NewRepository(db)
	lh := httphandler.NewLocateHandler(repo, httphandler.NewHandler(nil, false))
	mux := http.NewServeMux()
	lh.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/locate/next", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("locate next (empty) status = %d, want 404", rec.Code)
	}
}
