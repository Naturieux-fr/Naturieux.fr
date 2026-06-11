package quiz_test

import (
	"testing"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

func TestQuestionAccessors(t *testing.T) {
	correct, _ := species.New(1, "Sp one", "Un", "Aves")
	wrong, _ := species.New(2, "Sp two", "Deux", "Aves")
	q, err := quiz.NewQuestion("q1", quiz.FlashQuiz, quiz.Expert, correct,
		[]quiz.Choice{{Species: correct, IsCorrect: true}, {Species: wrong}}, "https://x/p.jpg")
	if err != nil {
		t.Fatalf("NewQuestion: %v", err)
	}
	if q.QuizType() != quiz.FlashQuiz || q.Difficulty() != quiz.Expert {
		t.Errorf("type/diff = %s/%s", q.QuizType(), q.Difficulty())
	}
	if q.MediaURL() != "https://x/p.jpg" {
		t.Errorf("media = %q", q.MediaURL())
	}
	q.SetMediaZoom(&species.PhotoRegion{X: 0.1, Y: 0.1, W: 0.5, H: 0.5})
	if q.MediaZoom() == nil {
		t.Error("zoom not set")
	}
	if q.TimeLimit() <= 0 || q.FlashDuration() <= 0 {
		t.Errorf("limits = %v / %v", q.TimeLimit(), q.FlashDuration())
	}
	if !q.CheckAnswer(1) || q.CheckAnswer(2) {
		t.Error("CheckAnswer wrong")
	}
}
