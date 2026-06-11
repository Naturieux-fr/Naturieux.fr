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
	// CreateQuestion generates a question of the given type and difficulty,
	// restricted to taxonFilter (a category); an empty taxonFilter falls back
	// to the factory's default.
	CreateQuestion(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty, taxonFilter string) (*quiz.Question, error)
	// CreateQuestionFor builds a question for a specific species (curated quizzes).
	CreateQuestionFor(ctx context.Context, quizType quiz.QuizType, difficulty quiz.Difficulty, speciesID int) (*quiz.Question, error)
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
	taxonFilter string,
) (*quiz.Question, error) {
	// The per-session category takes precedence over the factory default.
	taxon := taxonFilter
	if taxon == "" {
		taxon = f.taxonFilter
	}

	// Get random species for the correct answer. The difficulty is passed
	// through so sources that tag photos by difficulty (TAXREF) can prefer
	// a photo matching the session level.
	filter := ports.SpeciesFilter{
		IconicTaxon: taxon,
		PlaceID:     f.placeID,
		Limit:       1,
		HasPhotos:   true,
		Difficulty:  string(difficulty),
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
	return f.buildQuestion(ctx, quizType, difficulty, correct)
}

// CreateQuestionFor builds a question whose correct answer is the given species
// (used by curated challenges that pin a specific list of cd_noms).
func (f *questionFactory) CreateQuestionFor(
	ctx context.Context,
	quizType quiz.QuizType,
	difficulty quiz.Difficulty,
	speciesID int,
) (*quiz.Question, error) {
	correct, err := f.speciesRepo.GetByID(ctx, speciesID)
	if err != nil {
		return nil, fmt.Errorf("getting species %d: %w", speciesID, err)
	}
	if !correct.HasPhotos() {
		return nil, fmt.Errorf("species %d has no photos", speciesID)
	}
	return f.buildQuestion(ctx, quizType, difficulty, correct)
}

// buildQuestion assembles a question around a chosen correct species: it draws
// same-group distractors, shuffles the choices, and attaches the media credit.
func (f *questionFactory) buildQuestion(
	ctx context.Context,
	quizType quiz.QuizType,
	difficulty quiz.Difficulty,
	correct *species.Species,
) (*quiz.Question, error) {
	config := quiz.DefaultDifficultyConfigs()[difficulty]

	wrongChoices, err := f.getWrongChoices(ctx, correct, config.ChoicesCount-1)
	if err != nil {
		return nil, fmt.Errorf("getting wrong choices: %w", err)
	}

	choices := make([]quiz.Choice, 0, config.ChoicesCount)
	choices = append(choices, quiz.Choice{Species: correct, IsCorrect: true})
	for _, wrong := range wrongChoices {
		choices = append(choices, quiz.Choice{Species: wrong, IsCorrect: false})
	}
	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})

	mediaURL, mediaPhoto := f.selectMedia(correct, quizType)
	question, err := quiz.NewQuestion(uuid.New().String(), quizType, difficulty, correct, choices, mediaURL)
	if err != nil {
		return nil, err
	}
	if mediaPhoto != nil {
		question.SetMediaCredit(mediaPhoto.Attribution, mediaPhoto.LicenseCode)
		question.SetMediaZoom(mediaPhoto.Zoom)
	}
	return question, nil
}

// minDistractors is the fewest wrong answers a question can have: a question
// needs at least the correct answer plus one distractor.
const minDistractors = 1

// getWrongChoices retrieves species to use as incorrect answers. Distractors
// are always taxonomically related to the correct species (same genus, then
// family, then order, then group) and NEVER from another group — a frog quiz
// must never offer a snake. When the group is too small to fill every slot,
// the question simply has fewer choices.
func (f *questionFactory) getWrongChoices(
	ctx context.Context,
	correct *species.Species,
	count int,
) ([]*species.Species, error) {
	similar := f.fetchSimilarSpecies(ctx, correct.ID(), count)
	if len(similar) >= count {
		return similar[:count], nil
	}

	// Top up with other species from the same group (same iconic taxon).
	random, err := f.fetchRandomSpecies(ctx, correct, count)
	if err != nil {
		return nil, err
	}

	result := f.combineUniqueSpecies(correct.ID(), similar, random, count)
	if len(result) < minDistractors {
		return nil, errors.New("not enough related species for choices")
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

// fetchRandomSpecies retrieves random species from the same group as the
// correct answer. Distractors only display a name, so no photo is required —
// this keeps the pool of plausible same-group wrong answers as wide as
// possible.
func (f *questionFactory) fetchRandomSpecies(
	ctx context.Context,
	correct *species.Species,
	count int,
) ([]*species.Species, error) {
	filter := ports.SpeciesFilter{
		IconicTaxon: correct.IconicTaxon(),
		Limit:       count + 5,
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

// selectMedia selects the appropriate media URL and source photo based on
// quiz type. The photo carries the attribution and license that must be
// displayed alongside the media.
func (f *questionFactory) selectMedia(sp *species.Species, quizType quiz.QuizType) (string, *species.Photo) {
	photos := sp.Photos()
	if len(photos) == 0 {
		return "", nil
	}

	photo := photos[0]

	switch quizType {
	case quiz.ImageQuiz:
		return firstNonEmpty(photo.LargeURL, photo.MediumURL, photo.URL), &photo
	case quiz.FlashQuiz:
		// Use medium for faster loading
		return firstNonEmpty(photo.MediumURL, photo.URL), &photo
	case quiz.PartialQuiz, quiz.SilhouetteQuiz:
		// Use original for processing
		return firstNonEmpty(photo.OriginalURL, photo.LargeURL, photo.URL), &photo
	case quiz.SoundQuiz:
		// Sound quiz uses audio, not photos - return empty
		return "", nil
	}

	// Default fallback (should not be reached with exhaustive switch)
	return firstNonEmpty(photo.MediumURL, photo.URL), &photo
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
