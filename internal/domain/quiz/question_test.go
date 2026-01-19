package quiz_test

import (
	"testing"
	"time"

	"github.com/fieve/naturieux/internal/domain/quiz"
	"github.com/fieve/naturieux/internal/domain/species"
)

func createTestSpecies(id int, name string) *species.Species {
	s, _ := species.New(id, name, name, "Mammalia")
	s.AddPhoto(species.Photo{ID: 1, URL: "https://example.com/photo.jpg"})
	return s
}

func TestNewQuestion(t *testing.T) {
	correct := createTestSpecies(1, "Vulpes vulpes")
	wrong1 := createTestSpecies(2, "Vulpes zerda")
	wrong2 := createTestSpecies(3, "Canis lupus")

	choices := []quiz.Choice{
		{Species: correct, IsCorrect: true},
		{Species: wrong1, IsCorrect: false},
		{Species: wrong2, IsCorrect: false},
	}

	tests := []struct {
		name       string
		id         string
		quizType   quiz.QuizType
		difficulty quiz.Difficulty
		species    *species.Species
		choices    []quiz.Choice
		mediaURL   string
		wantErr    bool
	}{
		{
			name:       "valid question",
			id:         "q1",
			quizType:   quiz.ImageQuiz,
			difficulty: quiz.Beginner,
			species:    correct,
			choices:    choices,
			mediaURL:   "https://example.com/image.jpg",
			wantErr:    false,
		},
		{
			name:       "missing id",
			id:         "",
			quizType:   quiz.ImageQuiz,
			difficulty: quiz.Beginner,
			species:    correct,
			choices:    choices,
			mediaURL:   "https://example.com/image.jpg",
			wantErr:    true,
		},
		{
			name:       "invalid quiz type",
			id:         "q1",
			quizType:   "invalid",
			difficulty: quiz.Beginner,
			species:    correct,
			choices:    choices,
			mediaURL:   "https://example.com/image.jpg",
			wantErr:    true,
		},
		{
			name:       "missing species",
			id:         "q1",
			quizType:   quiz.ImageQuiz,
			difficulty: quiz.Beginner,
			species:    nil,
			choices:    choices,
			mediaURL:   "https://example.com/image.jpg",
			wantErr:    true,
		},
		{
			name:       "not enough choices",
			id:         "q1",
			quizType:   quiz.ImageQuiz,
			difficulty: quiz.Beginner,
			species:    correct,
			choices:    []quiz.Choice{{Species: correct, IsCorrect: true}},
			mediaURL:   "https://example.com/image.jpg",
			wantErr:    true,
		},
		{
			name:       "no correct answer in choices",
			id:         "q1",
			quizType:   quiz.ImageQuiz,
			difficulty: quiz.Beginner,
			species:    correct,
			choices: []quiz.Choice{
				{Species: wrong1, IsCorrect: false},
				{Species: wrong2, IsCorrect: false},
			},
			mediaURL: "https://example.com/image.jpg",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := quiz.NewQuestion(
				tt.id, tt.quizType, tt.difficulty,
				tt.species, tt.choices, tt.mediaURL,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewQuestion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && q == nil {
				t.Error("NewQuestion() returned nil without error")
			}
		})
	}
}

func TestQuestion_CheckAnswer(t *testing.T) {
	correct := createTestSpecies(1, "Vulpes vulpes")
	wrong := createTestSpecies(2, "Vulpes zerda")

	choices := []quiz.Choice{
		{Species: correct, IsCorrect: true},
		{Species: wrong, IsCorrect: false},
	}

	q, _ := quiz.NewQuestion("q1", quiz.ImageQuiz, quiz.Beginner, correct, choices, "https://example.com/image.jpg")

	if !q.CheckAnswer(1) {
		t.Error("CheckAnswer(1) = false, want true for correct answer")
	}

	if q.CheckAnswer(2) {
		t.Error("CheckAnswer(2) = true, want false for wrong answer")
	}
}

func TestQuestion_CalculateScore(t *testing.T) {
	correct := createTestSpecies(1, "Vulpes vulpes")
	wrong := createTestSpecies(2, "Vulpes zerda")

	choices := []quiz.Choice{
		{Species: correct, IsCorrect: true},
		{Species: wrong, IsCorrect: false},
	}

	tests := []struct {
		name       string
		difficulty quiz.Difficulty
		timeTaken  time.Duration
		isCorrect  bool
		wantMin    int
		wantMax    int
	}{
		{
			name:       "beginner fast correct",
			difficulty: quiz.Beginner,
			timeTaken:  5 * time.Second,
			isCorrect:  true,
			wantMin:    120,
			wantMax:    150,
		},
		{
			name:       "beginner slow correct",
			difficulty: quiz.Beginner,
			timeTaken:  29 * time.Second,
			isCorrect:  true,
			wantMin:    100,
			wantMax:    105,
		},
		{
			name:       "wrong answer",
			difficulty: quiz.Beginner,
			timeTaken:  5 * time.Second,
			isCorrect:  false,
			wantMin:    0,
			wantMax:    0,
		},
		{
			name:       "expert fast correct",
			difficulty: quiz.Expert,
			timeTaken:  5 * time.Second,
			isCorrect:  true,
			wantMin:    200,
			wantMax:    300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, _ := quiz.NewQuestion("q1", quiz.ImageQuiz, tt.difficulty, correct, choices, "https://example.com/image.jpg")
			score := q.CalculateScore(tt.timeTaken, tt.isCorrect)
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("CalculateScore() = %d, want between %d and %d", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestQuizType_IsValid(t *testing.T) {
	validTypes := []quiz.QuizType{
		quiz.ImageQuiz, quiz.FlashQuiz, quiz.PartialQuiz,
		quiz.SilhouetteQuiz, quiz.SoundQuiz,
	}

	for _, qt := range validTypes {
		if !quiz.IsValidQuizType(qt) {
			t.Errorf("IsValidQuizType(%s) = false, want true", qt)
		}
	}

	if quiz.IsValidQuizType("invalid") {
		t.Error("IsValidQuizType(invalid) = true, want false")
	}
}

func TestDifficulty_IsValid(t *testing.T) {
	validDiffs := []quiz.Difficulty{
		quiz.Beginner, quiz.Intermediate, quiz.Expert, quiz.Master,
	}

	for _, d := range validDiffs {
		if !quiz.IsValidDifficulty(d) {
			t.Errorf("IsValidDifficulty(%s) = false, want true", d)
		}
	}

	if quiz.IsValidDifficulty("invalid") {
		t.Error("IsValidDifficulty(invalid) = true, want false")
	}
}
