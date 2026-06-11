package quiz_test

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

func newQuizService(t *testing.T) *appquiz.Service {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return appquiz.NewService(
		appquiz.NewQuestionFactory(mock.NewSpeciesRepository()),
		sqlite.NewSessionRepository(db),
		sqlite.NewPlayerRepository(db),
		nil,
	)
}

func TestService_PlayersSessionsRevision(t *testing.T) {
	s := newQuizService(t)
	ctx := context.Background()

	p, err := s.RegisterPlayer(ctx, "Bob")
	if err != nil {
		t.Fatalf("RegisterPlayer: %v", err)
	}
	if got, err := s.GetPlayer(ctx, p.ID()); err != nil || got.ID() != p.ID() {
		t.Errorf("GetPlayer = %v, %v", got, err)
	}
	if _, err := s.GetLeaderboard(ctx, 10); err != nil {
		t.Errorf("GetLeaderboard: %v", err)
	}

	// Start a normal session, then look it up.
	start, err := s.StartSession(ctx, appquiz.StartSessionRequest{
		UserID: p.ID(), Difficulty: quiz.Beginner, QuizTypes: []quiz.QuizType{quiz.ImageQuiz}, QuestionCount: 2,
	})
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if sess, err := s.GetSession(ctx, start.SessionID); err != nil || sess == nil {
		t.Errorf("GetSession: %v", err)
	}

	// Revision from a couple of species.
	if _, err := s.StartRevisionSession(ctx, p.ID(), []int{1, 2}); err != nil {
		t.Errorf("StartRevisionSession: %v", err)
	}
}
