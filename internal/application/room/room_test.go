package room_test

import (
	"context"
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

// fakeMaker builds simple questions whose correct answer is species id 1.
type fakeMaker struct{}

func (fakeMaker) CreateQuestion(_ context.Context, qt quiz.QuizType, d quiz.Difficulty, _ string) (*quiz.Question, error) {
	if d == "" {
		d = quiz.Beginner
	}
	correct, _ := species.New(1, "Correct sp", "La Bonne", "Mammifères")
	wrong, _ := species.New(2, "Wrong sp", "La Mauvaise", "Mammifères")
	for _, s := range []*species.Species{correct, wrong} {
		s.AddPhoto(species.Photo{MediumURL: "u", LicenseCode: "cc-by"})
	}
	return quiz.NewQuestion("q", qt, d, correct, []quiz.Choice{
		{Species: correct, IsCorrect: true},
		{Species: wrong, IsCorrect: false},
	}, "u")
}

func TestRoom_FullGame(t *testing.T) {
	m := room.NewManager(fakeMaker{})
	ctx := context.Background()

	r, hostTok, err := m.Create("h", "Hôte", room.Settings{Count: 3, Difficulty: quiz.Beginner})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	code := r.Code()

	st, _ := m.Snapshot(code)
	if st.Status != room.Lobby || len(st.Players) != 1 {
		t.Fatalf("lobby state = %+v", st)
	}

	_, rivalTok, err := m.Join(code, "p2", "Rival")
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}

	// Only the host's token can start.
	if err := m.Start(ctx, code, rivalTok); err != room.ErrForbidden {
		t.Errorf("Start(non-host) = %v, want ErrForbidden", err)
	}
	if err := m.Start(ctx, code, hostTok); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// A bogus token cannot answer.
	if _, err := m.Answer(code, "not-a-token", 1); err != room.ErrPlayerUnknown {
		t.Errorf("Answer(bad token) = %v, want ErrPlayerUnknown", err)
	}

	// Host answers all 3 correctly (builds a streak), rival answers wrong.
	for i := 0; i < 3; i++ {
		res, err := m.Answer(code, hostTok, 1)
		if err != nil {
			t.Fatalf("Answer(host) error = %v", err)
		}
		if !res.IsCorrect {
			t.Error("host answer should be correct")
		}
		if _, err := m.Answer(code, rivalTok, 2); err != nil {
			t.Fatalf("Answer(rival) error = %v", err)
		}
	}

	final, _ := m.Snapshot(code)
	if final.Status != room.Done {
		t.Errorf("status = %s, want finished", final.Status)
	}
	if final.Players[0].ID != "h" || final.Players[0].Rank != 1 {
		t.Errorf("rank 1 = %+v, want host", final.Players[0])
	}
	if final.Players[0].Best < 3 {
		t.Errorf("host best streak = %d, want >= 3", final.Players[0].Best)
	}
	if final.Players[0].Score <= final.Players[1].Score {
		t.Error("host score should beat the rival's")
	}
}

func TestRoom_Elimination(t *testing.T) {
	m := room.NewManager(fakeMaker{})
	ctx := context.Background()
	r, tok, _ := m.Create("h", "Hôte", room.Settings{Count: 5, Mode: room.Elimination})
	code := r.Code()
	if err := m.Start(ctx, code, tok); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// One wrong answer ends the game immediately (sudden death).
	res, err := m.Answer(code, tok, 999) // wrong species id
	if err != nil {
		t.Fatalf("Answer() error = %v", err)
	}
	if !res.Eliminated || !res.Done {
		t.Errorf("expected elimination, got %+v", res)
	}
	st, _ := m.Snapshot(code)
	if !st.Players[0].Eliminated {
		t.Error("player should be flagged eliminated in the snapshot")
	}
}

func TestRoom_GuessName(t *testing.T) {
	m := room.NewManager(fakeMaker{})
	ctx := context.Background()
	r, tok, _ := m.Create("h", "Hôte", room.Settings{Count: 2, AnswerMode: "free"})
	code := r.Code()
	_ = m.Start(ctx, code, tok)

	if ok, _, _ := m.GuessName(code, tok, "nimporte quoi"); ok {
		t.Error("wrong guess should not match")
	}
	ok, id, err := m.GuessName(code, tok, "La Bonne")
	if err != nil || !ok || id != 1 {
		t.Errorf("GuessName(correct) = (%v, %d, %v), want (true, 1, nil)", ok, id, err)
	}
}

func TestRoom_JoinUnknown(t *testing.T) {
	m := room.NewManager(fakeMaker{})
	if _, _, err := m.Join("ZZZZ", "p", "X"); err != room.ErrNotFound {
		t.Errorf("Join(unknown) = %v, want ErrNotFound", err)
	}
}
