package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestPlayerRepository_CreateAndGet(t *testing.T) {
	repo := sqlite.NewPlayerRepository(openTestDB(t))
	ctx := context.Background()

	player, _ := gamification.NewPlayer("p1", "alice")
	player.AddXP(150)

	if err := repo.Create(ctx, player); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, "p1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Username() != "alice" {
		t.Errorf("Username = %s, want alice", got.Username())
	}
	if got.TotalXP() != 150 {
		t.Errorf("TotalXP = %d, want 150", got.TotalXP())
	}
	if got.Level() != player.Level() {
		t.Errorf("Level = %d, want %d", got.Level(), player.Level())
	}

	byName, err := repo.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if byName.ID() != "p1" {
		t.Errorf("GetByUsername ID = %s, want p1", byName.ID())
	}
}

func TestPlayerRepository_GetByID_NotFound(t *testing.T) {
	repo := sqlite.NewPlayerRepository(openTestDB(t))

	_, err := repo.GetByID(context.Background(), "ghost")
	if !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestPlayerRepository_Update(t *testing.T) {
	repo := sqlite.NewPlayerRepository(openTestDB(t))
	ctx := context.Background()

	player, _ := gamification.NewPlayer("p1", "alice")
	if err := repo.Create(ctx, player); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	player.RecordGame(8, 10, 5)
	player.AddXP(300)
	if err := repo.Update(ctx, player); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, "p1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.TotalGames() != 1 {
		t.Errorf("TotalGames = %d, want 1", got.TotalGames())
	}
	if got.BestStreak() != 5 {
		t.Errorf("BestStreak = %d, want 5", got.BestStreak())
	}
	if len(got.Achievements()) != len(player.Achievements()) {
		t.Errorf("Achievements = %d, want %d", len(got.Achievements()), len(player.Achievements()))
	}

	ghost, _ := gamification.NewPlayer("ghost", "ghost")
	if err := repo.Update(ctx, ghost); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("Update(ghost) error = %v, want ErrNotFound", err)
	}
}

func TestPlayerRepository_GetLeaderboard(t *testing.T) {
	repo := sqlite.NewPlayerRepository(openTestDB(t))
	ctx := context.Background()

	for _, p := range []struct {
		id, name string
		xp       int
	}{
		{"p1", "alice", 100},
		{"p2", "bob", 500},
		{"p3", "carol", 300},
	} {
		player, _ := gamification.NewPlayer(p.id, p.name)
		player.AddXP(p.xp)
		if err := repo.Create(ctx, player); err != nil {
			t.Fatalf("Create(%s) error = %v", p.id, err)
		}
	}

	top, err := repo.GetLeaderboard(ctx, 2)
	if err != nil {
		t.Fatalf("GetLeaderboard() error = %v", err)
	}
	if len(top) != 2 {
		t.Fatalf("Leaderboard size = %d, want 2", len(top))
	}
	if top[0].Username() != "bob" || top[1].Username() != "carol" {
		t.Errorf("Leaderboard order = %s, %s; want bob, carol", top[0].Username(), top[1].Username())
	}
}

func buildTestSession(t *testing.T, userID string) *quiz.Session {
	t.Helper()

	makeSpecies := func(id int, name string) *species.Species {
		sp, err := species.New(id, name, name+" common", "Aves")
		if err != nil {
			t.Fatalf("species.New() error = %v", err)
		}
		sp.AddPhoto(species.Photo{
			ID:          id,
			MediumURL:   "https://example.com/photo.jpg",
			Attribution: "(c) Someone, CC BY",
			LicenseCode: "cc-by",
		})
		return sp
	}

	correct := makeSpecies(1, "Correct")
	question, err := quiz.NewQuestion("q1", quiz.ImageQuiz, quiz.Beginner, correct, []quiz.Choice{
		{Species: correct, IsCorrect: true},
		{Species: makeSpecies(2, "Wrong"), IsCorrect: false},
	}, "https://example.com/photo.jpg")
	if err != nil {
		t.Fatalf("NewQuestion() error = %v", err)
	}
	question.SetMediaCredit("(c) Someone, CC BY", "cc-by")

	session, err := quiz.NewSessionBuilder().
		WithUserID(userID).
		WithDifficulty(quiz.Beginner).
		WithQuizTypes(quiz.ImageQuiz).
		WithTaxonFilter("Aves").
		WithQuestions([]*quiz.Question{question}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	return session
}

func TestSessionRepository_SaveAndGet(t *testing.T) {
	repo := sqlite.NewSessionRepository(openTestDB(t))
	ctx := context.Background()

	session := buildTestSession(t, "u1")
	if err := session.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := repo.Save(ctx, session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.GetByID(ctx, session.ID())
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.UserID() != "u1" {
		t.Errorf("UserID = %s, want u1", got.UserID())
	}
	if got.Status() != quiz.SessionInProgress {
		t.Errorf("Status = %s, want in_progress", got.Status())
	}

	// The restored session must be playable: answer the question
	answer, err := got.SubmitAnswer(1, 5*time.Second)
	if err != nil {
		t.Fatalf("SubmitAnswer() on restored session error = %v", err)
	}
	if !answer.IsCorrect {
		t.Error("SubmitAnswer() should be correct for species 1")
	}
	if got.Status() != quiz.SessionCompleted {
		t.Errorf("Status after last answer = %s, want completed", got.Status())
	}

	// Persist the update and check it sticks
	if err := repo.Save(ctx, got); err != nil {
		t.Fatalf("Save(updated) error = %v", err)
	}
	final, err := repo.GetByID(ctx, session.ID())
	if err != nil {
		t.Fatalf("GetByID(final) error = %v", err)
	}
	if final.Status() != quiz.SessionCompleted {
		t.Errorf("Final status = %s, want completed", final.Status())
	}
	if final.CorrectCount() != 1 {
		t.Errorf("CorrectCount = %d, want 1", final.CorrectCount())
	}

	// The restored question keeps its media credit
	snap := final.Snapshot()
	if snap.Questions[0].MediaAttribution == "" {
		t.Error("Restored question lost its media attribution")
	}
}

func TestSessionRepository_GetByID_NotFound(t *testing.T) {
	repo := sqlite.NewSessionRepository(openTestDB(t))

	_, err := repo.GetByID(context.Background(), "ghost")
	if !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestSessionRepository_GetByUserID(t *testing.T) {
	repo := sqlite.NewSessionRepository(openTestDB(t))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		session := buildTestSession(t, "u1")
		_ = session.Start()
		if err := repo.Save(ctx, session); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}
	other := buildTestSession(t, "u2")
	_ = other.Start()
	_ = repo.Save(ctx, other)

	sessions, err := repo.GetByUserID(ctx, "u1", 2)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Sessions = %d, want 2 (limit)", len(sessions))
	}
	for _, s := range sessions {
		if s.UserID() != "u1" {
			t.Errorf("UserID = %s, want u1", s.UserID())
		}
	}
}

func TestSessionRepository_GetStats(t *testing.T) {
	repo := sqlite.NewSessionRepository(openTestDB(t))
	ctx := context.Background()

	// Two completed sessions with one correct answer each
	for i := 0; i < 2; i++ {
		session := buildTestSession(t, "u1")
		_ = session.Start()
		if _, err := session.SubmitAnswer(1, 5*time.Second); err != nil {
			t.Fatalf("SubmitAnswer() error = %v", err)
		}
		if err := repo.Save(ctx, session); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}
	// One abandoned session that must not count
	abandoned := buildTestSession(t, "u1")
	_ = abandoned.Start()
	abandoned.Abandon()
	_ = repo.Save(ctx, abandoned)

	stats, err := repo.GetStats(ctx, "u1")
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	if stats.TotalSessions != 2 {
		t.Errorf("TotalSessions = %d, want 2", stats.TotalSessions)
	}
	if stats.TotalCorrect != 2 {
		t.Errorf("TotalCorrect = %d, want 2", stats.TotalCorrect)
	}
	if stats.AverageAccuracy != 100 {
		t.Errorf("AverageAccuracy = %f, want 100", stats.AverageAccuracy)
	}
	if stats.FavoriteTaxon != "Aves" {
		t.Errorf("FavoriteTaxon = %s, want Aves", stats.FavoriteTaxon)
	}
}
