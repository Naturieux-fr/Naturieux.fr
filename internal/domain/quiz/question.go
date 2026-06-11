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
	id               string
	quizType         QuizType
	difficulty       Difficulty
	correctSpecies   *species.Species
	choices          []Choice
	mediaURL         string
	mediaAttribution string
	mediaLicense     string
	mediaZoom        *species.PhotoRegion
	timeLimit        time.Duration
	flashDuration    time.Duration
	createdAt        time.Time
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

// SetMediaCredit records the author attribution and license of the media.
func (q *Question) SetMediaCredit(attribution, licenseCode string) {
	q.mediaAttribution = attribution
	q.mediaLicense = licenseCode
}

// MediaAttribution returns the author attribution of the media.
func (q *Question) MediaAttribution() string {
	return q.mediaAttribution
}

// MediaLicense returns the license code of the media (e.g. "cc-by-nc").
func (q *Question) MediaLicense() string {
	return q.mediaLicense
}

// SetMediaZoom records the area the Détail mode should crop to.
func (q *Question) SetMediaZoom(z *species.PhotoRegion) {
	q.mediaZoom = z
}

// MediaZoom returns the Détail-mode crop region, or nil.
func (q *Question) MediaZoom() *species.PhotoRegion {
	return q.mediaZoom
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

// ChoiceSnapshot is a serializable representation of a Choice.
type ChoiceSnapshot struct {
	Species   species.Snapshot `json:"species"`
	IsCorrect bool             `json:"is_correct"`
}

// QuestionSnapshot is a serializable representation of a Question, used for
// persistence and reconstruction.
type QuestionSnapshot struct {
	ID               string               `json:"id"`
	QuizType         QuizType             `json:"quiz_type"`
	Difficulty       Difficulty           `json:"difficulty"`
	Choices          []ChoiceSnapshot     `json:"choices"`
	MediaURL         string               `json:"media_url"`
	MediaAttribution string               `json:"media_attribution,omitempty"`
	MediaLicense     string               `json:"media_license,omitempty"`
	MediaZoom        *species.PhotoRegion `json:"media_zoom,omitempty"`
}

// Snapshot returns a serializable representation of the question.
func (q *Question) Snapshot() QuestionSnapshot {
	choices := make([]ChoiceSnapshot, len(q.choices))
	for i, c := range q.choices {
		choices[i] = ChoiceSnapshot{
			Species:   c.Species.Snapshot(),
			IsCorrect: c.IsCorrect,
		}
	}
	return QuestionSnapshot{
		ID:               q.id,
		QuizType:         q.quizType,
		Difficulty:       q.difficulty,
		Choices:          choices,
		MediaURL:         q.mediaURL,
		MediaAttribution: q.mediaAttribution,
		MediaLicense:     q.mediaLicense,
		MediaZoom:        q.mediaZoom,
	}
}

// RestoreQuestion reconstructs a Question from a snapshot.
func RestoreQuestion(snap QuestionSnapshot) (*Question, error) {
	choices := make([]Choice, len(snap.Choices))
	var correct *species.Species
	for i, c := range snap.Choices {
		sp, err := species.FromSnapshot(c.Species)
		if err != nil {
			return nil, err
		}
		choices[i] = Choice{Species: sp, IsCorrect: c.IsCorrect}
		if c.IsCorrect {
			correct = sp
		}
	}

	question, err := NewQuestion(snap.ID, snap.QuizType, snap.Difficulty, correct, choices, snap.MediaURL)
	if err != nil {
		return nil, err
	}
	question.SetMediaCredit(snap.MediaAttribution, snap.MediaLicense)
	question.SetMediaZoom(snap.MediaZoom)
	return question, nil
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
