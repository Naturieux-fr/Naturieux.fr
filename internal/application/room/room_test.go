package room_test

import (
	"context"
	"testing"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/application/room"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

// fakeMaker builds simple questions whose correct answer is species id 1.
type fakeMaker struct{}

func (fakeMaker) CreateQuestion(_ context.Context, qt quiz.QuizType, d quiz.Difficulty, _ string) (*quiz.Question, error) {
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

	r, err := m.Create("h", "Hôte", room.Settings{Count: 3, Difficulty: quiz.Beginner})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	st, _ := m.Snapshot(r.Code())
	if st.Status != room.Lobby || len(st.Players) != 1 {
		t.Fatalf("lobby state = %+v", st)
	}

	if _, err := m.Join(st.Code, "p2", "Rival"); err != nil {
		t.Fatalf("Join() error = %v", err)
	}

	// Only the host can start.
	if err := m.Start(ctx, st.Code, "p2"); err != room.ErrForbidden {
		t.Errorf("Start(non-host) = %v, want ErrForbidden", err)
	}
	if err := m.Start(ctx, st.Code, "h"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Host answers all 3 correctly (builds a streak), rival answers wrong.
	for i := 0; i < 3; i++ {
		res, err := m.Answer(st.Code, "h", 1, time.Second)
		if err != nil {
			t.Fatalf("Answer(host) error = %v", err)
		}
		if !res.IsCorrect {
			t.Error("host answer should be correct")
		}
		if _, err := m.Answer(st.Code, "p2", 2, time.Second); err != nil {
			t.Fatalf("Answer(rival) error = %v", err)
		}
	}

	final, _ := m.Snapshot(st.Code)
	if final.Status != room.Done {
		t.Errorf("status = %s, want finished", final.Status)
	}
	// Host (all correct) must rank first with a streak.
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

func TestRoom_JoinUnknown(t *testing.T) {
	m := room.NewManager(fakeMaker{})
	if _, err := m.Join("ZZZZ", "p", "X"); err != room.ErrNotFound {
		t.Errorf("Join(unknown) = %v, want ErrNotFound", err)
	}
}
