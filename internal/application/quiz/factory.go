// Package quiz contains application services for quiz management.
package quiz

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/species"
	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
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

// Minimum number of choices required for a valid question.
const minChoicesRequired = 2

// getWrongChoices retrieves species to use as incorrect answers.
func (f *questionFactory) getWrongChoices(
	ctx context.Context,
	correct *species.Species,
	count int,
) ([]*species.Species, error) {
	similar := f.fetchSimilarSpecies(ctx, correct.ID(), count)
	if len(similar) >= count {
		return similar[:count], nil
	}

	random, err := f.fetchRandomSpecies(ctx, correct, count)
	if err != nil {
		return nil, err
	}

	result := f.combineUniqueSpecies(correct.ID(), similar, random, count)
	if len(result) < minChoicesRequired {
		return nil, errors.New("not enough species for choices")
	}

	return result, nil
}

// fetchSimilarSpecies retrieves similar species, returning empty slice on error.
func (f *questionFactory) fetchSimilarSpecies(ctx context.Context, speciesID, count int) []*species.Species {
	similar, err := f.speciesRepo.GetSimilar(ctx, speciesID, count)
	if err != nil {
		return nil
	}
	return similar
}

// fetchRandomSpecies retrieves random species from the same taxon.
func (f *questionFactory) fetchRandomSpecies(ctx context.Context, correct *species.Species, count int) ([]*species.Species, error) {
	filter := ports.SpeciesFilter{
		IconicTaxon: correct.IconicTaxon(),
		Limit:       count + 5,
		HasPhotos:   true,
		ExcludeIDs:  []int{correct.ID()},
	}
	return f.speciesRepo.GetRandom(ctx, filter)
}

// combineUniqueSpecies merges species lists, removing duplicates and the correct species.
func (f *questionFactory) combineUniqueSpecies(
	correctID int,
	similar, random []*species.Species,
	maxCount int,
) []*species.Species {
	seen := map[int]bool{correctID: true}
	result := make([]*species.Species, 0, maxCount)

	result = collectUniqueSpecies(result, similar, seen, maxCount)
	result = collectUniqueSpecies(result, random, seen, maxCount)

	return result
}

// collectUniqueSpecies adds unique species to the result slice up to maxCount.
func collectUniqueSpecies(
	result, candidates []*species.Species,
	seen map[int]bool,
	maxCount int,
) []*species.Species {
	for _, sp := range candidates {
		if len(result) >= maxCount {
			break
		}
		if !seen[sp.ID()] {
			seen[sp.ID()] = true
			result = append(result, sp)
		}
	}
	return result
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
