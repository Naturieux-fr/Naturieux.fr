package quiz_test

import (
	"testing"
	"time"

	"github.com/fieve/naturieux/internal/domain/quiz"
	"github.com/fieve/naturieux/internal/domain/species"
)

func createTestQuestion(id string, correctID int) *quiz.Question {
	correct, _ := species.New(correctID, "Correct Species", "Correct", "Mammalia")
	correct.AddPhoto(species.Photo{ID: 1, URL: "https://example.com/photo.jpg"})

	wrong, _ := species.New(correctID+100, "Wrong Species", "Wrong", "Mammalia")

	choices := []quiz.Choice{
		{Species: correct, IsCorrect: true},
		{Species: wrong, IsCorrect: false},
	}

	q, _ := quiz.NewQuestion(id, quiz.ImageQuiz, quiz.Beginner, correct, choices, "https://example.com/img.jpg")
	return q
}

func TestSessionBuilder(t *testing.T) {
	q1 := createTestQuestion("q1", 1)
	q2 := createTestQuestion("q2", 2)

	tests := []struct {
		name      string
		userID    string
		questions []*quiz.Question
		wantErr   bool
	}{
		{
			name:      "valid session",
			userID:    "user1",
			questions: []*quiz.Question{q1, q2},
			wantErr:   false,
		},
		{
			name:      "missing user id",
			userID:    "",
			questions: []*quiz.Question{q1},
			wantErr:   true,
		},
		{
			name:      "no questions",
			userID:    "user1",
			questions: []*quiz.Question{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := quiz.NewSessionBuilder().
				WithUserID(tt.userID).
				WithDifficulty(quiz.Beginner).
				WithQuizTypes(quiz.ImageQuiz).
				WithQuestions(tt.questions).
				Build()

			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && session == nil {
				t.Error("Build() returned nil without error")
			}
		})
	}
}

func TestSession_Start(t *testing.T) {
	q := createTestQuestion("q1", 1)
	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions([]*quiz.Question{q}).
		Build()

	if session.Status() != quiz.SessionPending {
		t.Errorf("Status() = %v, want pending", session.Status())
	}

	err := session.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if session.Status() != quiz.SessionInProgress {
		t.Errorf("Status() = %v, want in_progress", session.Status())
	}

	// Second start should fail
	err = session.Start()
	if err == nil {
		t.Error("Start() should fail when already started")
	}
}

func TestSession_SubmitAnswer(t *testing.T) {
	q1 := createTestQuestion("q1", 1)
	q2 := createTestQuestion("q2", 2)

	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions([]*quiz.Question{q1, q2}).
		Build()

	// Submit before start should fail
	_, err := session.SubmitAnswer(1, 5*time.Second)
	if err == nil {
		t.Error("SubmitAnswer() should fail before start")
	}

	session.Start()

	// Correct answer
	answer, err := session.SubmitAnswer(1, 5*time.Second)
	if err != nil {
		t.Errorf("SubmitAnswer() error = %v", err)
	}
	if !answer.IsCorrect {
		t.Error("Answer should be correct")
	}
	if answer.Score <= 0 {
		t.Error("Score should be positive for correct answer")
	}

	// Wrong answer
	answer, err = session.SubmitAnswer(999, 5*time.Second)
	if err != nil {
		t.Errorf("SubmitAnswer() error = %v", err)
	}
	if answer.IsCorrect {
		t.Error("Answer should be wrong")
	}
	if answer.Score != 0 {
		t.Errorf("Score should be 0 for wrong answer, got %d", answer.Score)
	}

	// Session should be completed
	if session.Status() != quiz.SessionCompleted {
		t.Errorf("Status() = %v, want completed", session.Status())
	}
}

func TestSession_Streak(t *testing.T) {
	questions := make([]*quiz.Question, 5)
	for i := 0; i < 5; i++ {
		questions[i] = createTestQuestion("q"+string(rune('1'+i)), i+1)
	}

	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions(questions).
		Build()
	session.Start()

	// Answer 3 correct in a row
	for i := 1; i <= 3; i++ {
		session.SubmitAnswer(i, 5*time.Second)
	}

	if session.CurrentStreak() != 3 {
		t.Errorf("CurrentStreak() = %d, want 3", session.CurrentStreak())
	}

	// Wrong answer breaks streak
	session.SubmitAnswer(999, 5*time.Second)

	if session.CurrentStreak() != 0 {
		t.Errorf("CurrentStreak() = %d, want 0 after wrong answer", session.CurrentStreak())
	}

	if session.MaxStreak() != 3 {
		t.Errorf("MaxStreak() = %d, want 3", session.MaxStreak())
	}
}

func TestSession_Accuracy(t *testing.T) {
	questions := make([]*quiz.Question, 4)
	for i := 0; i < 4; i++ {
		questions[i] = createTestQuestion("q"+string(rune('1'+i)), i+1)
	}

	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions(questions).
		Build()
	session.Start()

	// 2 correct, 2 wrong
	session.SubmitAnswer(1, 5*time.Second)   // correct
	session.SubmitAnswer(2, 5*time.Second)   // correct
	session.SubmitAnswer(999, 5*time.Second) // wrong
	session.SubmitAnswer(999, 5*time.Second) // wrong

	if session.Accuracy() != 50 {
		t.Errorf("Accuracy() = %f, want 50", session.Accuracy())
	}

	if session.CorrectCount() != 2 {
		t.Errorf("CorrectCount() = %d, want 2", session.CorrectCount())
	}
}

func TestSession_Abandon(t *testing.T) {
	q := createTestQuestion("q1", 1)
	session, _ := quiz.NewSessionBuilder().
		WithUserID("user1").
		WithQuestions([]*quiz.Question{q}).
		Build()
	session.Start()

	session.Abandon()

	if session.Status() != quiz.SessionAbandoned {
		t.Errorf("Status() = %v, want abandoned", session.Status())
	}
}
