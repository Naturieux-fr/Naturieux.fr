package quiz_test

import (
	"testing"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

func makeSnapshotTestSpecies(t *testing.T, id int, name string) *species.Species {
	t.Helper()
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

func makeSnapshotTestQuestion(t *testing.T) *quiz.Question {
	t.Helper()
	correct := makeSnapshotTestSpecies(t, 1, "Correct")
	question, err := quiz.NewQuestion("q1", quiz.ImageQuiz, quiz.Beginner, correct, []quiz.Choice{
		{Species: correct, IsCorrect: true},
		{Species: makeSnapshotTestSpecies(t, 2, "Wrong"), IsCorrect: false},
	}, "https://example.com/photo.jpg")
	if err != nil {
		t.Fatalf("NewQuestion() error = %v", err)
	}
	question.SetMediaCredit("(c) Someone, CC BY", "cc-by")
	return question
}

func TestQuestion_SnapshotRoundTrip(t *testing.T) {
	question := makeSnapshotTestQuestion(t)

	restored, err := quiz.RestoreQuestion(question.Snapshot())
	if err != nil {
		t.Fatalf("RestoreQuestion() error = %v", err)
	}

	if restored.ID() != question.ID() {
		t.Errorf("ID = %s, want %s", restored.ID(), question.ID())
	}
	if restored.CorrectSpecies().ID() != 1 {
		t.Errorf("CorrectSpecies ID = %d, want 1", restored.CorrectSpecies().ID())
	}
	if len(restored.Choices()) != 2 {
		t.Errorf("Choices = %d, want 2", len(restored.Choices()))
	}
	if restored.MediaAttribution() != "(c) Someone, CC BY" {
		t.Errorf("MediaAttribution = %s, lost in round trip", restored.MediaAttribution())
	}
	if restored.MediaLicense() != "cc-by" {
		t.Errorf("MediaLicense = %s, want cc-by", restored.MediaLicense())
	}
	if !restored.CheckAnswer(1) {
		t.Error("CheckAnswer(1) should be true on restored question")
	}
}

func TestRestoreQuestion_InvalidSpecies(t *testing.T) {
	snap := quiz.QuestionSnapshot{
		ID:         "q1",
		QuizType:   quiz.ImageQuiz,
		Difficulty: quiz.Beginner,
		MediaURL:   "https://example.com/photo.jpg",
		Choices: []quiz.ChoiceSnapshot{
			{Species: species.Snapshot{ID: 0}, IsCorrect: true},
			{Species: species.Snapshot{ID: 0}, IsCorrect: false},
		},
	}
	if _, err := quiz.RestoreQuestion(snap); err == nil {
		t.Error("RestoreQuestion() with invalid species should return an error")
	}
}

func TestSession_SnapshotRoundTrip(t *testing.T) {
	question := makeSnapshotTestQuestion(t)
	session, err := quiz.NewSessionBuilder().
		WithUserID("u1").
		WithDifficulty(quiz.Beginner).
		WithQuizTypes(quiz.ImageQuiz).
		WithTaxonFilter("Aves").
		WithQuestions([]*quiz.Question{question}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := session.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	restored, err := quiz.RestoreSession(session.Snapshot())
	if err != nil {
		t.Fatalf("RestoreSession() error = %v", err)
	}

	if restored.ID() != session.ID() {
		t.Errorf("ID = %s, want %s", restored.ID(), session.ID())
	}
	if restored.TaxonFilter() != "Aves" {
		t.Errorf("TaxonFilter = %s, want Aves", restored.TaxonFilter())
	}
	if restored.Status() != quiz.SessionInProgress {
		t.Errorf("Status = %s, want in_progress", restored.Status())
	}
	if restored.StartedAt().IsZero() {
		t.Error("StartedAt should be preserved")
	}
	if restored.CompletedAt() != nil {
		t.Error("CompletedAt should be nil for a running session")
	}

	// Answer on the restored session: it completes and records the answer
	if _, err := restored.SubmitAnswer(1, 3*time.Second); err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}
	if restored.Status() != quiz.SessionCompleted {
		t.Errorf("Status = %s, want completed", restored.Status())
	}

	// Round trip again with answers recorded
	final, err := quiz.RestoreSession(restored.Snapshot())
	if err != nil {
		t.Fatalf("RestoreSession(answered) error = %v", err)
	}
	if final.AnsweredCount() != 1 {
		t.Errorf("AnsweredCount = %d, want 1", final.AnsweredCount())
	}
	if final.CurrentQuestion() != nil {
		t.Error("CurrentQuestion should be nil once all questions are answered")
	}
	if final.TotalScore() != restored.TotalScore() {
		t.Errorf("TotalScore = %d, want %d", final.TotalScore(), restored.TotalScore())
	}
}

func TestRestoreSession_Invalid(t *testing.T) {
	cases := []struct {
		name string
		snap quiz.SessionSnapshot
	}{
		{"missing id", quiz.SessionSnapshot{UserID: "u1"}},
		{"missing user", quiz.SessionSnapshot{ID: "s1"}},
		{"no questions", quiz.SessionSnapshot{ID: "s1", UserID: "u1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := quiz.RestoreSession(tc.snap); err == nil {
				t.Error("RestoreSession() should return an error")
			}
		})
	}
}
