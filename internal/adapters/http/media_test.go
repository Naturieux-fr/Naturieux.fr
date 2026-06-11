package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
	adminapp "github.com/Naturieux-fr/Naturieux.fr/internal/application/admin"
)

func TestAdminMediaAndLocate(t *testing.T) {
	db := memDB(t)
	ctx := context.Background()
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("taxref schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO taxref_species
		(cd_nom, cd_ref, rang, scientific_name, vernacular_name, kingdom, class, ordre, family, genus, taxa_group, fr) VALUES
		(1,1,'ES','Erithacus rubecula','Rougegorge','Animalia','Aves','Passeriformes','Muscicapidae','Erithacus','Oiseaux','P'),
		(2,2,'ES','Turdus merula','Merle','Animalia','Aves','Passeriformes','Turdidae','Turdus','Oiseaux','P')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	taxrefRepo := taxref.NewRepository(db)
	store, _ := storage.NewLocal(t.TempDir())
	playerRepo := sqlite.NewPlayerRepository(db)
	authSvc := adminapp.NewService(playerRepo, "secret")
	_ = authSvc.SeedAdmin(ctx, "boss", "bosspass1")
	accSvc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Open)

	adminH := httphandler.NewAdminHandler(authSvc, taxrefRepo, store, accSvc)
	mux := http.NewServeMux()
	adminH.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "boss", "password": "bosspass1"}))
	_, d := decode(t, rec)
	token := d["token"].(string)

	send := func(method, path string, body any) *httptest.ResponseRecorder {
		var r *http.Request
		if s, ok := body.(string); ok {
			r = httptest.NewRequest(method, path, strings.NewReader(s))
		} else {
			r = jsonReq(method, path, body)
		}
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	if w := send(http.MethodGet, "/api/v1/admin/taxa?q=Erithacus", nil); w.Code != http.StatusOK {
		t.Errorf("search = %d: %s", w.Code, w.Body)
	}
	w := send(http.MethodPost, "/api/v1/admin/taxa/1/photos", map[string]any{"url": "https://x/p.jpg", "attribution": "a", "license": "cc-by"})
	if w.Code != http.StatusOK {
		t.Fatalf("add photo = %d: %s", w.Code, w.Body)
	}
	_, pd := decode(t, w)
	pid := int(pd["id"].(float64))

	if w := send(http.MethodGet, "/api/v1/admin/taxa/1/photos", nil); w.Code != http.StatusOK {
		t.Errorf("list photos = %d", w.Code)
	}
	zones := `{"zoom":{"x":0.1,"y":0.1,"w":0.5,"h":0.5},"species":[{"cd_nom":1,"name":"R","x":0.05,"y":0.05,"w":0.9,"h":0.9}]}`
	if w := send(http.MethodPost, "/api/v1/admin/photos/"+strconv.Itoa(pid)+"/zones", zones); w.Code != http.StatusOK {
		t.Errorf("set zones = %d: %s", w.Code, w.Body)
	}
	if w := send(http.MethodGet, "/api/v1/admin/coverage", nil); w.Code != http.StatusOK {
		t.Errorf("coverage = %d", w.Code)
	}
	if w := send(http.MethodPost, "/api/v1/admin/taxa/2/sounds", map[string]any{"url": "https://x/s.mp3"}); w.Code != http.StatusOK {
		t.Errorf("add sound = %d: %s", w.Code, w.Body)
	}

	// Locate flow against the same annotated photo.
	locateH := httphandler.NewLocateHandler(taxrefRepo, httphandler.NewHandler(nil, false))
	lmux := http.NewServeMux()
	locateH.RegisterRoutes(lmux)

	rec = httptest.NewRecorder()
	lmux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/locate/next", nil))
	ok, nd := decode(t, rec)
	if !ok {
		t.Fatalf("locate next: %s", rec.Body)
	}
	rec = httptest.NewRecorder()
	lmux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/locate/answer", map[string]any{
		"photo_id": int(nd["photo_id"].(float64)), "cd_nom": int(nd["target_cd"].(float64)), "x": 0.5, "y": 0.5,
	}))
	if ok, ad := decode(t, rec); !ok || ad["correct"] != true {
		t.Errorf("locate answer (centre) = %s", rec.Body)
	}
}
