package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
)

func TestConfig_RegistrationMode(t *testing.T) {
	h := httphandler.NewHandler(nil, true)
	h.SetRegistrationMode("invite")
	rec := httptest.NewRecorder()
	h.HandleConfig(rec, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	_, d := decode(t, rec)
	if d["registration_mode"] != "invite" {
		t.Errorf("registration_mode = %v", d["registration_mode"])
	}
}

func TestAccount_InviteModeRejectsAnon(t *testing.T) {
	db := memDB(t)
	playerRepo := sqlite.NewPlayerRepository(db)
	svc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Invite)
	h := httphandler.NewAccountHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/account/register", map[string]any{"username": "Eve", "password": "secret1"}))
	if rec.Code != http.StatusForbidden {
		t.Errorf("invite-only register = %d, want 403", rec.Code)
	}
}

func TestLocate_ImageLocalMedia(t *testing.T) {
	db := memDB(t)
	ctx := context.Background()
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	_, _ = db.Exec(`INSERT INTO taxref_species (cd_nom,cd_ref,rang,scientific_name,kingdom,taxa_group,fr) VALUES (1,1,'ES','Sp one','Animalia','Oiseaux','P')`)
	repo := taxref.NewRepository(db)
	pid, _ := repo.AddPhoto(ctx, 1, "/media/zz.jpg", "a", "cc-by", "")
	_ = repo.SetPhotoZones(ctx, pid, `{"species":[{"cd_nom":1,"name":"S","x":0,"y":0,"w":1,"h":1}]}`)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "zz.jpg"), []byte("IMG"), 0o600)
	images := httphandler.NewHandler(nil, false)
	images.SetLocalMediaDir(dir)

	lh := httphandler.NewLocateHandler(repo, images)
	mux := http.NewServeMux()
	lh.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/locate/image/"+itoa(pid), nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "IMG" {
		t.Errorf("locate image = %d / %q", rec.Code, rec.Body.String())
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
