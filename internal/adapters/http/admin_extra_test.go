package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
	adminapp "github.com/Naturieux-fr/Naturieux.fr/internal/application/admin"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
)

type fakeRooms struct{}

func (fakeRooms) Count() int { return 2 }

func TestAdminHandlers(t *testing.T) {
	db := memDB(t)
	ctx := context.Background()
	playerRepo := sqlite.NewPlayerRepository(db)

	authSvc := adminapp.NewService(playerRepo, "secret")
	if err := authSvc.SeedAdmin(ctx, "boss", "bosspass1"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	accSvc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Open)
	accSvc.SetResetStore(sqlite.NewResetRepository(db))
	curated := sqlite.NewCuratedRepository(db)
	mgr := challenge.NewManager(appquiz.NewQuestionFactory(mock.NewSpeciesRepository()), curated)

	h := httphandler.NewAdminHandler(authSvc, nil, nil, accSvc)
	h.SetAdminData(playerRepo, fakeRooms{})
	h.SetCuratedData(curated, mgr)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// A second player to promote / reset.
	member, _, _ := accSvc.Register(ctx, "Member", "secret1", "")

	// Login as admin.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "boss", "password": "bosspass1"}))
	ok, data := decode(t, rec)
	if !ok {
		t.Fatalf("admin login: %s", rec.Body)
	}
	token := data["token"].(string)
	auth := func(method, path string, body any) *httptest.ResponseRecorder {
		r := jsonReq(method, path, body)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	// Unauthenticated is rejected.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anon stats = %d", rec.Code)
	}

	if w := auth(http.MethodGet, "/api/v1/admin/stats", nil); w.Code != http.StatusOK {
		t.Errorf("stats = %d: %s", w.Code, w.Body)
	}
	if w := auth(http.MethodGet, "/api/v1/admin/players", nil); w.Code != http.StatusOK {
		t.Errorf("players = %d", w.Code)
	}

	// Invitations.
	if w := auth(http.MethodPost, "/api/v1/admin/invites", map[string]any{}); w.Code != http.StatusOK {
		t.Errorf("create invite = %d: %s", w.Code, w.Body)
	}
	if w := auth(http.MethodGet, "/api/v1/admin/invites", nil); w.Code != http.StatusOK {
		t.Errorf("list invites = %d", w.Code)
	}

	// Curated quiz + schedule.
	w := auth(http.MethodPost, "/api/v1/admin/quizzes", map[string]any{"name": "Q", "species": []int{1, 2}})
	if w.Code != http.StatusOK {
		t.Fatalf("create quiz = %d: %s", w.Code, w.Body)
	}
	_, qd := decode(t, w)
	qid := qd["id"].(string)
	if w := auth(http.MethodPost, "/api/v1/admin/challenge/schedule", map[string]any{"period": "daily", "quiz_id": qid}); w.Code != http.StatusOK {
		t.Errorf("schedule = %d: %s", w.Code, w.Body)
	}

	// Promote member to writer and issue a reset link.
	if w := auth(http.MethodPost, "/api/v1/admin/players/"+member.ID()+"/role", map[string]any{"role": "writer"}); w.Code != http.StatusOK {
		t.Errorf("set role = %d: %s", w.Code, w.Body)
	}
	if w := auth(http.MethodPost, "/api/v1/admin/players/"+member.ID()+"/reset", map[string]any{}); w.Code != http.StatusOK {
		t.Errorf("reset = %d: %s", w.Code, w.Body)
	}

	// Demoting the only admin is blocked.
	if w := auth(http.MethodPost, "/api/v1/admin/players/"+findAdminID(t, playerRepo)+"/role", map[string]any{"role": "player"}); w.Code != http.StatusConflict {
		t.Errorf("demote last admin = %d, want 409", w.Code)
	}
}

func findAdminID(t *testing.T, r *sqlite.PlayerRepository) string {
	t.Helper()
	list, _ := r.ListPlayers(context.Background(), 50)
	for _, p := range list {
		if p.Role == "admin" {
			return p.ID
		}
	}
	t.Fatal("no admin found")
	return ""
}
