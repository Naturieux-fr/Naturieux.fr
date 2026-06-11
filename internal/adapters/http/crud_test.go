package http_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
	adminapp "github.com/Naturieux-fr/Naturieux.fr/internal/application/admin"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
)

func multipartFile(field, filename string, data []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fw, _ := w.CreateFormFile(field, filename)
	_, _ = fw.Write(data)
	_ = w.WriteField("attribution", "a")
	_ = w.WriteField("license", "cc-by")
	_ = w.Close()
	return body, w.FormDataContentType()
}

func TestAdminCRUDAndUploads(t *testing.T) {
	db := memDB(t)
	ctx := context.Background()
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	_, _ = db.Exec(`INSERT INTO taxref_species (cd_nom,cd_ref,rang,scientific_name,kingdom,taxa_group,fr) VALUES (1,1,'ES','Erithacus rubecula','Animalia','Oiseaux','P')`)
	taxrefRepo := taxref.NewRepository(db)
	store, _ := storage.NewLocal(t.TempDir())
	playerRepo := sqlite.NewPlayerRepository(db)
	authSvc := adminapp.NewService(playerRepo, "secret")
	_ = authSvc.SeedAdmin(ctx, "boss", "bosspass1")
	accSvc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Open)
	curated := sqlite.NewCuratedRepository(db)
	mgr := challenge.NewManager(appquiz.NewQuestionFactory(taxrefRepo), curated)

	adminH := httphandler.NewAdminHandler(authSvc, taxrefRepo, store, accSvc)
	adminH.SetAdminData(playerRepo, fakeRooms{})
	adminH.SetCuratedData(curated, mgr)
	articleH := httphandler.NewArticleHandler(sqlite.NewArticleRepository(db), accSvc)
	accH := httphandler.NewAccountHandler(accSvc)
	mux := http.NewServeMux()
	adminH.RegisterRoutes(mux)
	articleH.RegisterRoutes(mux)
	accH.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "boss", "password": "bosspass1"}))
	_, ld := decode(t, rec)
	token := ld["token"].(string)
	send := func(method, path string, body any) *httptest.ResponseRecorder {
		r := jsonReq(method, path, body)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	upload := func(path, filename string, data []byte) *httptest.ResponseRecorder {
		body, ct := multipartFile("file", filename, data)
		r := httptest.NewRequest(http.MethodPost, path, body)
		r.Header.Set("Content-Type", ct)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	// Invite create + revoke.
	_, inv := decode(t, send(http.MethodPost, "/api/v1/admin/invites", map[string]any{}))
	if w := send(http.MethodPost, "/api/v1/admin/invites/"+inv["invite"].(string)+"/revoke", map[string]any{}); w.Code != http.StatusOK {
		t.Errorf("revoke invite = %d: %s", w.Code, w.Body)
	}

	// Quiz create -> list -> delete.
	_, q := decode(t, send(http.MethodPost, "/api/v1/admin/quizzes", map[string]any{"name": "Q", "species": []int{1}}))
	if w := send(http.MethodGet, "/api/v1/admin/quizzes", nil); w.Code != http.StatusOK {
		t.Errorf("list quizzes = %d", w.Code)
	}
	if w := send(http.MethodDelete, "/api/v1/admin/quizzes/"+q["id"].(string), nil); w.Code != http.StatusOK {
		t.Errorf("delete quiz = %d", w.Code)
	}

	// Multipart photo upload (JPEG magic) + sound upload (ID3 magic).
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 16)...)
	if w := upload("/api/v1/admin/taxa/1/upload", "p.jpg", jpeg); w.Code != http.StatusOK {
		t.Errorf("photo upload = %d: %s", w.Code, w.Body)
	}
	mp3 := append([]byte("ID3"), make([]byte, 16)...)
	if w := upload("/api/v1/admin/taxa/1/sound-upload", "s.mp3", mp3); w.Code != http.StatusOK {
		t.Errorf("sound upload = %d: %s", w.Code, w.Body)
	}
	// Sounds list + delete.
	sid, _ := taxrefRepo.AddSound(ctx, 1, "https://x/s2.mp3", "r", "cc-by")
	if w := send(http.MethodGet, "/api/v1/admin/taxa/1/sounds", nil); w.Code != http.StatusOK {
		t.Errorf("list sounds = %d", w.Code)
	}
	if w := send(http.MethodDelete, "/api/v1/admin/sounds/"+strconv.Itoa(sid), nil); w.Code != http.StatusOK {
		t.Errorf("delete sound = %d", w.Code)
	}

	// Register a member, delete them.
	member, _, _ := accSvc.Register(ctx, "Gone", "secret1", "")
	if w := send(http.MethodDelete, "/api/v1/admin/players/"+member.ID(), nil); w.Code != http.StatusOK {
		t.Errorf("delete player = %d", w.Code)
	}

	// Article get + delete (writer).
	writer, wtok, _ := accSvc.Register(ctx, "Writer", "secret1", "")
	_ = playerRepo.SetRole(ctx, writer.ID(), "writer")
	wr := jsonReq(http.MethodPost, "/api/v1/articles", map[string]any{"title": "T", "body": "B", "published": false})
	wr.Header.Set("Authorization", "Bearer "+wtok)
	arec := httptest.NewRecorder()
	mux.ServeHTTP(arec, wr)
	_, ad := decode(t, arec)
	aid := ad["id"].(string)
	// Draft is hidden from anonymous.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/articles/"+aid, nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("anon draft = %d, want 404", rec.Code)
	}
	// Author can delete it.
	dr := httptest.NewRequest(http.MethodDelete, "/api/v1/articles/"+aid, nil)
	dr.Header.Set("Authorization", "Bearer "+wtok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, dr)
	if rec.Code != http.StatusOK {
		t.Errorf("delete article = %d: %s", rec.Code, rec.Body)
	}
}
