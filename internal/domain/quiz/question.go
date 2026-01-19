package quiz

import (
	"errors"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
)

// Choice represents an answer choice for a question.
type Choice struct {
	Species   *species.Species
	IsCorrect bool
}

// Question represents a single quiz question.
type Question struct {
	id             string
	quizType       QuizType
	difficulty     Difficulty
	correctSpecies *species.Species
	choices        []Choice
	mediaURL       string
	timeLimit      time.Duration
	flashDuration  time.Duration
	createdAt      time.Time
}

// NewQuestion creates a new Question with validation.
func NewQuestion(
	id string,
	quizType QuizType,
	difficulty Difficulty,
	correctSpecies *species.Species,
	choices []Choice,
	mediaURL string,
) (*Question, error) {
	if id == "" {
		return nil, errors.New("question id is required")
	}
	if !IsValidQuizType(quizType) {
		return nil, errors.New("invalid quiz type")
	}
	if !IsValidDifficulty(difficulty) {
		return nil, errors.New("invalid difficulty")
	}
	if correctSpecies == nil {
		return nil, errors.New("correct species is required")
	}
	if len(choices) < 2 {
		return nil, errors.New("at least 2 choices are required")
	}
	if mediaURL == "" {
		return nil, errors.New("media URL is required")
	}

	// Verify correct species is in choices
	hasCorrect := false
	for _, c := range choices {
		if c.IsCorrect {
			hasCorrect = true
			break
		}
	}
	if !hasCorrect {
		return nil, errors.New("choices must contain correct answer")
	}

	config := DefaultDifficultyConfigs()[difficulty]

	return &Question{
		id:             id,
		quizType:       quizType,
		difficulty:     difficulty,
		correctSpecies: correctSpecies,
		choices:        choices,
		mediaURL:       mediaURL,
		timeLimit:      config.TimeLimit,
		flashDuration:  config.FlashDuration,
		createdAt:      time.Now(),
	}, nil
}

// ID returns the question ID.
func (q *Question) ID() string {
	return q.id
}

// QuizType returns the quiz type.
func (q *Question) QuizType() QuizType {
	return q.quizType
}

// Difficulty returns the difficulty level.
func (q *Question) Difficulty() Difficulty {
	return q.difficulty
}

// CorrectSpecies returns the correct species.
func (q *Question) CorrectSpecies() *species.Species {
	return q.correctSpecies
}

// Choices returns all answer choices.
func (q *Question) Choices() []Choice {
	return q.choices
}

// MediaURL returns the media URL for the question.
func (q *Question) MediaURL() string {
	return q.mediaURL
}

// TimeLimit returns the time limit for answering.
func (q *Question) TimeLimit() time.Duration {
	return q.timeLimit
}

// FlashDuration returns how long media is shown for FlashQuiz.
func (q *Question) FlashDuration() time.Duration {
	return q.flashDuration
}

// CheckAnswer verifies if the given species ID is correct.
func (q *Question) CheckAnswer(speciesID int) bool {
	return q.correctSpecies.ID() == speciesID
}

// CalculateScore calculates score based on time taken and difficulty.
func (q *Question) CalculateScore(timeTaken time.Duration, isCorrect bool) int {
	if !isCorrect {
		return 0
	}

	baseScore := 100
	config := DefaultDifficultyConfigs()[q.difficulty]

	// Time bonus: faster = more points
	timeRatio := float64(q.timeLimit-timeTaken) / float64(q.timeLimit)
	if timeRatio < 0 {
		timeRatio = 0
	}
	timeBonus := int(float64(baseScore) * timeRatio * 0.5)

	return int(float64(baseScore+timeBonus) * config.ScoreMultiplier)
}
