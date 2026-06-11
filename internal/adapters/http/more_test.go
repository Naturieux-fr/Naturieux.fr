package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/account"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
)

func TestAuthedQuizRevisionGuessChallenge(t *testing.T) {
	db := memDB(t)
	playerRepo := sqlite.NewPlayerRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)
	factory := appquiz.NewQuestionFactory(mock.NewSpeciesRepository())
	quizSvc := appquiz.NewService(factory, sessionRepo, playerRepo, nil)

	acc := account.NewService(playerRepo, playerRepo, sqlite.NewInviteRepository(db), "secret", account.Open)
	h := httphandler.NewHandler(quizSvc, false)
	h.SetAuthenticator(acc)
	h.SetMissTracker(sqlite.NewMissRepository(db))
	mgr := challenge.NewManager(factory, sqlite.NewCuratedRepository(db))
	ch := httphandler.NewChallengeHandler(mgr, quizSvc, sqlite.NewChallengeRepository(db), playerRepo)
	ch.SetAuthenticator(acc)
	accH := httphandler.NewAccountHandler(acc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ch.RegisterRoutes(mux)
	accH.RegisterRoutes(mux)

	// Register -> token.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, jsonReq(http.MethodPost, "/api/v1/account/register", map[string]any{"username": "Zoe", "password": "secret1"}))
	_, rd := decode(t, rec)
	token := rd["token"].(string)
	send := func(method, path string, body any) *httptest.ResponseRecorder {
		r := jsonReq(method, path, body)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	// Start a solo quiz, then answer (recording the outcome -> miss tracking).
	w := send(http.MethodPost, "/api/v1/quiz/start", map[string]any{"difficulty": "beginner", "quiz_types": []string{"image"}, "question_count": 3})
	if w.Code != http.StatusOK {
		t.Fatalf("start = %d: %s", w.Code, w.Body)
	}
	_, sd := decode(t, w)
	sid := sd["session_id"].(string)
	q := sd["question"].(map[string]any)
	choices := q["choices"].([]any)
	// Deliberately wrong answer (a far id) to record a miss.
	_ = send(http.MethodPost, "/api/v1/quiz/answer", map[string]any{"session_id": sid, "species_id": 999999, "time_taken_ms": 100})

	// Guess endpoint (free-text) on the current question.
	name := choices[0].(map[string]any)["display_name"].(string)
	if w := send(http.MethodPost, "/api/v1/quiz/"+sid+"/guess", map[string]any{"guess": name}); w.Code != http.StatusOK {
		t.Errorf("guess = %d: %s", w.Code, w.Body)
	}

	// Abandon the session.
	if w := send(http.MethodPost, "/api/v1/quiz/abandon", map[string]any{"session_id": sid}); w.Code != http.StatusOK {
		t.Errorf("abandon = %d: %s", w.Code, w.Body)
	}

	// Revision now has at least one missed species.
	if w := send(http.MethodPost, "/api/v1/quiz/revision", map[string]any{}); w.Code != http.StatusOK {
		t.Errorf("revision = %d: %s", w.Code, w.Body)
	}

	// Daily challenge: play to completion, then finish.
	w = send(http.MethodPost, "/api/v1/challenge/daily/start", map[string]any{})
	if w.Code != http.StatusOK {
		t.Fatalf("challenge start = %d: %s", w.Code, w.Body)
	}
	_, cs := decode(t, w)
	csid := cs["start"].(map[string]any)["session_id"].(string)
	cq := cs["start"].(map[string]any)["question"].(map[string]any)
	for i := 0; i < 15; i++ {
		ch := cq["choices"].([]any)
		ans := send(http.MethodPost, "/api/v1/quiz/answer", map[string]any{
			"session_id": csid, "species_id": int(ch[0].(map[string]any)["species_id"].(float64)), "time_taken_ms": 100,
		})
		_, ad := decode(t, ans)
		if done, _ := ad["session_complete"].(bool); done {
			break
		}
		if nq, ok := ad["next_question"].(map[string]any); ok {
			cq = nq
		}
	}
	if w := send(http.MethodPost, "/api/v1/challenge/finish", map[string]any{"session_id": csid}); w.Code != http.StatusOK {
		t.Errorf("challenge finish = %d: %s", w.Code, w.Body)
	}
}
