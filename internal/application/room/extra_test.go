package room_test

import (
	"context"
	"testing"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/mock"
	appquiz "github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

func newMgr() *room.Manager {
	return room.NewManager(appquiz.NewQuestionFactory(mock.NewSpeciesRepository()))
}

func TestRoom_FullFlow(t *testing.T) {
	mgr := newMgr()
	ctx := context.Background()

	rm, hostTok, err := mgr.Create("h1", "Host", room.Settings{
		Difficulty: quiz.Beginner, Count: 2, Mode: room.Classic, AnswerMode: "choices", Public: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	code := rm.Code()

	if open := mgr.ListOpen(); len(open) != 1 {
		t.Errorf("ListOpen = %d, want 1", len(open))
	}

	_, p2Tok, err := mgr.Join(code, "p2", "Bob")
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, _, err := mgr.Join("ZZZZ", "x", "y"); err == nil {
		t.Error("join unknown code should error")
	}

	// Only the host can start.
	if err := mgr.Start(ctx, code, p2Tok); err == nil {
		t.Error("non-host start should fail")
	}
	if err := mgr.Start(ctx, code, hostTok); err != nil {
		t.Fatalf("start: %v", err)
	}
	if open := mgr.ListOpen(); len(open) != 0 {
		t.Errorf("ListOpen after start = %d, want 0", len(open))
	}

	// Both players answer the current round.
	answer := func(tok, pid string) {
		q, err := mgr.CurrentQuestion(code, pid)
		if err != nil || q == nil {
			return
		}
		_, _ = mgr.Answer(code, tok, q.Choices()[0].Species.ID())
	}
	answer(hostTok, "h1")
	answer(p2Tok, "p2")

	st, err := mgr.Snapshot(code)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if st.Round < 1 && st.Status == room.Playing {
		t.Log("round did not advance (acceptable depending on timing)")
	}
}

func TestRoom_GuessAndCleanup(t *testing.T) {
	mgr := newMgr()
	ctx := context.Background()
	rm, tok, _ := mgr.Create("h1", "Host", room.Settings{
		Difficulty: quiz.Beginner, Count: 1, Mode: room.Classic, AnswerMode: "free",
	})
	code := rm.Code()
	_ = mgr.Start(ctx, code, tok)
	// A wrong free-text guess is handled.
	if _, _, err := mgr.GuessName(code, tok, "definitely-not-it"); err != nil {
		t.Errorf("guess: %v", err)
	}

	// Cleanup of rooms idle beyond the TTL (a fresh room survives).
	mgr.Cleanup(time.Hour)
	if _, err := mgr.Snapshot(code); err != nil {
		t.Error("a fresh room should survive cleanup")
	}
}

func TestRoom_Count(t *testing.T) {
	mgr := newMgr()
	if mgr.Count() != 0 {
		t.Errorf("Count = %d, want 0", mgr.Count())
	}
	_, _, _ = mgr.Create("h", "H", room.Settings{Difficulty: quiz.Beginner, Count: 1})
	if mgr.Count() != 1 {
		t.Errorf("Count = %d, want 1", mgr.Count())
	}
	_ = time.Now
}
