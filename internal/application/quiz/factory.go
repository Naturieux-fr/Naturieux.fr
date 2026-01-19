// Package quiz contains application services for quiz management.
package quiz

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/fieve/naturieux/internal/domain/quiz"
	"github.com/fieve/naturieux/internal/domain/species"
	"github.com/fieve/naturieux/internal/ports"
)

// QuestionFactory creates quiz questions of various types.
type QuestionFactory interface {
	// CreateQuestion generates a new question of the specified type and difficulty.
	CreateQuestion(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty) (*quiz.Question, error)
}

// questionFactory implements QuestionFactory.
type questionFactory struct {
	speciesRepo ports.SpeciesRepository
	taxonFilter string
	placeID     int
}

// QuestionFactoryOption configures the factory.
type QuestionFactoryOption func(*questionFactory)

// WithTaxonFilter sets the taxon filter for questions.
func WithTaxonFilter(taxon string) QuestionFactoryOption {
	return func(f *questionFactory) {
		f.taxonFilter = taxon
	}
}

// WithPlaceFilter sets the geographic filter for questions.
func WithPlaceFilter(placeID int) QuestionFactoryOption {
	return func(f *questionFactory) {
		f.placeID = placeID
	}
}

// NewQuestionFactory creates a new question factory.
func NewQuestionFactory(repo ports.SpeciesRepository, opts ...QuestionFactoryOption) QuestionFactory {
	f := &questionFactory{
		speciesRepo: repo,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// CreateQuestion generates a question of the specified type.
func (f *questionFactory) CreateQuestion(
	ctx context.Context,
	quizType quiz.QuizType,
	difficulty quiz.Difficulty,
) (*quiz.Question, error) {
	config := quiz.DefaultDifficultyConfigs()[difficulty]

	// Get random species for the correct answer
	filter := ports.SpeciesFilter{
		IconicTaxon: f.taxonFilter,
		PlaceID:     f.placeID,
		Limit:       1,
		HasPhotos:   true,
	}

	correctSpecies, err := f.speciesRepo.GetRandom(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("getting correct species: %w", err)
	}
	if len(correctSpecies) == 0 {
		return nil, errors.New("no species found matching criteria")
	}

	correct := correctSpecies[0]
	if !correct.HasPhotos() {
		return nil, errors.New("correct species has no photos")
	}

	// Get similar species for wrong answers
	wrongChoices, err := f.getWrongChoices(ctx, correct, config.ChoicesCount-1)
	if err != nil {
		return nil, fmt.Errorf("getting wrong choices: %w", err)
	}

	// Build choices
	choices := make([]quiz.Choice, 0, config.ChoicesCount)
	choices = append(choices, quiz.Choice{
		Species:   correct,
		IsCorrect: true,
	})
	for _, wrong := range wrongChoices {
		choices = append(choices, quiz.Choice{
			Species:   wrong,
			IsCorrect: false,
		})
	}

	// Shuffle choices
	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})

	// Get media URL
	mediaURL := f.selectMediaURL(correct, quizType)

	return quiz.NewQuestion(
		uuid.New().String(),
		quizType,
		difficulty,
		correct,
		choices,
		mediaURL,
	)
}

// getWrongChoices retrieves species to use as incorrect answers.
func (f *questionFactory) getWrongChoices(
	ctx context.Context,
	correct *species.Species,
	count int,
) ([]*species.Species, error) {
	// First try to get similar species (same genus/family)
	similar, err := f.speciesRepo.GetSimilar(ctx, correct.ID(), count)
	if err == nil && len(similar) >= count {
		return similar[:count], nil
	}

	// Fall back to random species from same taxon
	filter := ports.SpeciesFilter{
		IconicTaxon: correct.IconicTaxon(),
		Limit:       count + 5, // Get extra to ensure enough unique
		HasPhotos:   true,
		ExcludeIDs:  []int{correct.ID()},
	}

	random, err := f.speciesRepo.GetRandom(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Combine similar and random, remove duplicates
	seen := map[int]bool{correct.ID(): true}
	result := make([]*species.Species, 0, count)

	// Prefer similar species
	for _, sp := range similar {
		if !seen[sp.ID()] && len(result) < count {
			seen[sp.ID()] = true
			result = append(result, sp)
		}
	}

	// Fill with random if needed
	for _, sp := range random {
		if !seen[sp.ID()] && len(result) < count {
			seen[sp.ID()] = true
			result = append(result, sp)
		}
	}

	if len(result) < 2 {
		return nil, errors.New("not enough species for choices")
	}

	return result, nil
}

// selectMediaURL selects the appropriate media URL based on quiz type.
func (f *questionFactory) selectMediaURL(sp *species.Species, quizType quiz.QuizType) string {
	photos := sp.Photos()
	if len(photos) == 0 {
		return ""
	}

	photo := photos[0]

	switch quizType {
	case quiz.ImageQuiz:
		if photo.LargeURL != "" {
			return photo.LargeURL
		}
		return photo.MediumURL
	case quiz.FlashQuiz:
		// Use medium for faster loading
		return photo.MediumURL
	case quiz.PartialQuiz, quiz.SilhouetteQuiz:
		// Use original for processing
		if photo.OriginalURL != "" {
			return photo.OriginalURL
		}
		return photo.LargeURL
	case quiz.SoundQuiz:
		// Sound quiz uses audio, not photos - return empty
		return ""
	}

	// Default fallback (should not be reached with exhaustive switch)
	return photo.MediumURL
}
