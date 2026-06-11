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
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
)

func TestQuizImageProxy_LocalMedia(t *testing.T) {
	db := memDB(t)
	ctx := context.Background()
	if err := taxref.EnsureSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO taxref_species
		(cd_nom,cd_ref,rang,scientific_name,vernacular_name,kingdom,class,ordre,family,genus,taxa_group,fr) VALUES
		(1,1,'ES','Erithacus rubecula','Rougegorge','Animalia','Aves','Passeriformes','Muscicapidae','Erithacus','Oiseaux','P'),
		(2,2,'ES','Turdus merula','Merle','Animalia','Aves','Passeriformes','Turdidae','Turdus','Oiseaux','P')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "abc.jpg"), []byte("JPEGDATA"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	repo := taxref.NewRepository(db)
	if _, err := repo.AddPhoto(ctx, 1, "/media/abc.jpg", "a", "cc-by", "beginner"); err != nil {
		t.Fatalf("add photo: %v", err)
	}

	playerRepo := sqlite.NewPlayerRepository(db)
	p, _ := gamification.NewPlayer("u1", "U")
	_ = playerRepo.Create(ctx, p)

	quizSvc := appquiz.NewService(appquiz.NewQuestionFactory(repo), sqlite.NewSessionRepository(db), playerRepo, nil)
	h := httphandler.NewHandler(quizSvc, false)
	h.SetLocalMediaDir(dir)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/quiz/start", map[string]any{
		"user_id": "u1", "difficulty": "beginner", "quiz_types": []string{"image"}, "question_count": 1,
	}))
	ok, d := decode(t, rec)
	if !ok {
		t.Fatalf("start: %s", rec.Body)
	}
	sid := d["session_id"].(string)
	qid := d["question"].(map[string]any)["id"].(string)

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/quiz/"+sid+"/image?n="+qid, nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "JPEGDATA" {
		t.Errorf("image proxy = %d / %q", rec.Code, rec.Body.String())
	}
	// Wrong n -> not found.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/quiz/"+sid+"/image?n=wrong", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("wrong n = %d, want 404", rec.Code)
	}
}
