package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	adminapp "github.com/Naturieux-fr/Naturieux.fr/internal/application/admin"
)

const adminSampleTAXREF = "taxonID\tacceptedNameUsageID\tparentNameUsageID\tscientificName\tkingdom\tclass\torder\tfamily\tgenus\ttaxonRank\tvernacularName\n" +
	"60585\t60585\t198937\tVulpes vulpes\tAnimalia\tMammalia\tCarnivora\tCanidae\tVulpes\tspecies\tRenard roux\n"

// newAdminTest builds an admin handler backed by real SQLite + TAXREF, with a
// seeded admin account, and returns the handler plus a valid token.
func newAdminTest(t *testing.T) (*httphandler.AdminHandler, string) {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "admin.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	if _, err := taxref.Import(db, strings.NewReader(adminSampleTAXREF)); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	playerRepo := sqlite.NewPlayerRepository(db)
	authSvc := adminapp.NewService(playerRepo, "test-secret")
	if err := authSvc.SeedAdmin(context.Background(), "boss", "passw0rd!"); err != nil {
		t.Fatalf("SeedAdmin() error = %v", err)
	}
	handler := httphandler.NewAdminHandler(authSvc, taxref.NewRepository(db))

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Obtain a token via the login endpoint.
	body, _ := json.Marshal(map[string]string{"username": "boss", "password": "passw0rd!"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Data.Token == "" {
		t.Fatal("login returned empty token")
	}
	return handler, resp.Data.Token
}

// serve runs one request through the admin routes.
func serve(h *httphandler.AdminHandler, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAdmin_RejectsWithoutToken(t *testing.T) {
	handler, _ := newAdminTest(t)

	rec := serve(handler, httptest.NewRequest(http.MethodGet, "/api/v1/admin/taxa?q=fox", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no token status = %d, want 401", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/taxa?q=fox", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	rec = serve(handler, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("bad token status = %d, want 403", rec.Code)
	}
}

func TestAdmin_Login_WrongPassword(t *testing.T) {
	handler, _ := newAdminTest(t)
	body, _ := json.Marshal(map[string]string{"username": "boss", "password": "nope"})
	rec := serve(handler, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body)))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong password status = %d, want 401", rec.Code)
	}
}

func TestAdmin_PhotoLifecycle(t *testing.T) {
	handler, token := newAdminTest(t)
	auth := func(req *http.Request) *http.Request {
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}

	// Search finds the fox.
	rec := serve(handler, auth(httptest.NewRequest(http.MethodGet, "/api/v1/admin/taxa?q=Vulpes", nil)))
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d, want 200", rec.Code)
	}
	var search struct {
		Data struct {
			Taxa []map[string]interface{} `json:"taxa"`
		} `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&search)
	if len(search.Data.Taxa) != 1 {
		t.Fatalf("search returned %d taxa, want 1", len(search.Data.Taxa))
	}

	// Add a photo with a difficulty.
	body, _ := json.Marshal(map[string]string{
		"url": "https://photos.naturieux.fr/60585.jpg", "attribution": "(c) Moi",
		"license": "cc-by", "difficulty": "beginner",
	})
	rec = serve(handler, auth(httptest.NewRequest(http.MethodPost, "/api/v1/admin/taxa/60585/photos", bytes.NewReader(body))))
	if rec.Code != http.StatusOK {
		t.Fatalf("add photo status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var added struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&added)

	// List shows it.
	rec = serve(handler, auth(httptest.NewRequest(http.MethodGet, "/api/v1/admin/taxa/60585/photos", nil)))
	var listed struct {
		Data struct {
			Photos []taxref.PhotoRecord `json:"photos"`
		} `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&listed)
	if len(listed.Data.Photos) != 1 || listed.Data.Photos[0].Difficulty != "beginner" {
		t.Fatalf("list = %+v, want one beginner photo", listed.Data.Photos)
	}

	// Invalid difficulty is rejected.
	bad, _ := json.Marshal(map[string]string{"url": "u", "difficulty": "impossible"})
	rec = serve(handler, auth(httptest.NewRequest(http.MethodPost, "/api/v1/admin/taxa/60585/photos", bytes.NewReader(bad))))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad difficulty status = %d, want 400", rec.Code)
	}

	// Photo on an unknown taxon → 404.
	okBody, _ := json.Marshal(map[string]string{"url": "u"})
	rec = serve(handler, auth(httptest.NewRequest(http.MethodPost, "/api/v1/admin/taxa/999999/photos", bytes.NewReader(okBody))))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown taxon status = %d, want 404", rec.Code)
	}

	// Delete it.
	photoPath := "/api/v1/admin/photos/" + strconv.Itoa(added.Data.ID)
	rec = serve(handler, auth(httptest.NewRequest(http.MethodDelete, photoPath, nil)))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", rec.Code)
	}
	rec = serve(handler, auth(httptest.NewRequest(http.MethodDelete, photoPath, nil)))
	if rec.Code != http.StatusNotFound {
		t.Errorf("double delete status = %d, want 404", rec.Code)
	}
}
