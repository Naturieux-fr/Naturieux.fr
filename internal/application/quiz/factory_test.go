package quiz_test

import (
	"context"
	"errors"
	"testing"

	appquiz "github.com/fieve/naturieux/internal/application/quiz"
	"github.com/fieve/naturieux/internal/domain/quiz"
	"github.com/fieve/naturieux/internal/domain/species"
	"github.com/fieve/naturieux/internal/ports"
)

// mockSpeciesRepository is a test double for SpeciesRepository.
type mockSpeciesRepository struct {
	getByIDFunc    func(ctx context.Context, id int) (*species.Species, error)
	getRandomFunc  func(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error)
	getSimilarFunc func(ctx context.Context, speciesID int, limit int) ([]*species.Species, error)
	searchFunc     func(ctx context.Context, query string, limit int) ([]*species.Species, error)
}

func (m *mockSpeciesRepository) GetByID(ctx context.Context, id int) (*species.Species, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSpeciesRepository) GetRandom(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
	if m.getRandomFunc != nil {
		return m.getRandomFunc(ctx, filter)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSpeciesRepository) GetSimilar(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
	if m.getSimilarFunc != nil {
		return m.getSimilarFunc(ctx, speciesID, limit)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSpeciesRepository) Search(ctx context.Context, query string, limit int) ([]*species.Species, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, limit)
	}
	return nil, errors.New("not implemented")
}

func createMockSpecies(id int, name string) *species.Species {
	sp, _ := species.New(id, name, name+" Common", "Mammalia")
	sp.AddPhoto(species.Photo{
		ID:        id,
		URL:       "https://example.com/photo.jpg",
		MediumURL: "https://example.com/photo_medium.jpg",
		LargeURL:  "https://example.com/photo_large.jpg",
	})
	return sp
}

func TestQuestionFactory_CreateQuestion(t *testing.T) {
	correct := createMockSpecies(1, "Correct Species")
	wrong1 := createMockSpecies(2, "Wrong Species 1")
	wrong2 := createMockSpecies(3, "Wrong Species 2")
	wrong3 := createMockSpecies(4, "Wrong Species 3")

	mockRepo := &mockSpeciesRepository{
		getRandomFunc: func(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
			return []*species.Species{correct}, nil
		},
		getSimilarFunc: func(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
			return []*species.Species{wrong1, wrong2, wrong3}, nil
		},
	}

	factory := appquiz.NewQuestionFactory(mockRepo)

	question, err := factory.CreateQuestion(context.Background(), quiz.ImageQuiz, quiz.Beginner)
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}

	if question == nil {
		t.Fatal("CreateQuestion() returned nil")
	}

	if question.QuizType() != quiz.ImageQuiz {
		t.Errorf("QuizType = %v, want ImageQuiz", question.QuizType())
	}

	if question.Difficulty() != quiz.Beginner {
		t.Errorf("Difficulty = %v, want Beginner", question.Difficulty())
	}

	// Beginner should have 4 choices
	if len(question.Choices()) != 4 {
		t.Errorf("Choices count = %d, want 4", len(question.Choices()))
	}

	// Should have exactly one correct answer
	correctCount := 0
	for _, c := range question.Choices() {
		if c.IsCorrect {
			correctCount++
		}
	}
	if correctCount != 1 {
		t.Errorf("Correct answers = %d, want 1", correctCount)
	}
}

func TestQuestionFactory_CreateQuestion_NoSpeciesFound(t *testing.T) {
	mockRepo := &mockSpeciesRepository{
		getRandomFunc: func(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
			return []*species.Species{}, nil
		},
	}

	factory := appquiz.NewQuestionFactory(mockRepo)

	_, err := factory.CreateQuestion(context.Background(), quiz.ImageQuiz, quiz.Beginner)
	if err == nil {
		t.Error("CreateQuestion() should return error when no species found")
	}
}

func TestQuestionFactory_CreateQuestion_WithFilters(t *testing.T) {
	correct := createMockSpecies(1, "Test Bird")

	capturedFilter := ports.SpeciesFilter{}
	mockRepo := &mockSpeciesRepository{
		getRandomFunc: func(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
			capturedFilter = filter
			return []*species.Species{correct}, nil
		},
		getSimilarFunc: func(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
			return []*species.Species{
				createMockSpecies(2, "Wrong 1"),
				createMockSpecies(3, "Wrong 2"),
				createMockSpecies(4, "Wrong 3"),
			}, nil
		},
	}

	factory := appquiz.NewQuestionFactory(
		mockRepo,
		appquiz.WithTaxonFilter("Aves"),
		appquiz.WithPlaceFilter(6753), // France
	)

	_, err := factory.CreateQuestion(context.Background(), quiz.ImageQuiz, quiz.Beginner)
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}

	if capturedFilter.IconicTaxon != "Aves" {
		t.Errorf("IconicTaxon filter = %s, want Aves", capturedFilter.IconicTaxon)
	}

	if capturedFilter.PlaceID != 6753 {
		t.Errorf("PlaceID filter = %d, want 6753", capturedFilter.PlaceID)
	}
}

func TestQuestionFactory_CreateQuestion_ExpertDifficulty(t *testing.T) {
	correct := createMockSpecies(1, "Expert Species")
	wrongSpecies := make([]*species.Species, 10)
	for i := 0; i < 10; i++ {
		wrongSpecies[i] = createMockSpecies(i+2, "Wrong Species")
	}

	mockRepo := &mockSpeciesRepository{
		getRandomFunc: func(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
			return []*species.Species{correct}, nil
		},
		getSimilarFunc: func(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
			if limit > len(wrongSpecies) {
				return wrongSpecies, nil
			}
			return wrongSpecies[:limit], nil
		},
	}

	factory := appquiz.NewQuestionFactory(mockRepo)

	question, err := factory.CreateQuestion(context.Background(), quiz.ImageQuiz, quiz.Expert)
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}

	// Expert should have 8 choices
	if len(question.Choices()) != 8 {
		t.Errorf("Expert choices count = %d, want 8", len(question.Choices()))
	}
}

func TestQuestionFactory_CreateQuestion_SpeciesNoPhotos(t *testing.T) {
	// Species without photos
	speciesNoPhoto, _ := species.New(1, "No Photo Species", "Test", "Mammalia")

	mockRepo := &mockSpeciesRepository{
		getRandomFunc: func(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
			return []*species.Species{speciesNoPhoto}, nil
		},
	}

	factory := appquiz.NewQuestionFactory(mockRepo)

	_, err := factory.CreateQuestion(context.Background(), quiz.ImageQuiz, quiz.Beginner)
	if err == nil {
		t.Error("CreateQuestion() should return error when species has no photos")
	}
}

func TestQuestionFactory_CreateQuestion_FallbackToRandom(t *testing.T) {
	correct := createMockSpecies(1, "Correct Species")
	wrongRandom := createMockSpecies(2, "Random Wrong")

	mockRepo := &mockSpeciesRepository{
		getRandomFunc: func(ctx context.Context, filter ports.SpeciesFilter) ([]*species.Species, error) {
			if len(filter.ExcludeIDs) == 0 {
				return []*species.Species{correct}, nil
			}
			// Return random for wrong answers
			return []*species.Species{wrongRandom, createMockSpecies(3, "R2"), createMockSpecies(4, "R3")}, nil
		},
		getSimilarFunc: func(ctx context.Context, speciesID int, limit int) ([]*species.Species, error) {
			// Simulate no similar species found
			return nil, errors.New("no similar species")
		},
	}

	factory := appquiz.NewQuestionFactory(mockRepo)

	question, err := factory.CreateQuestion(context.Background(), quiz.ImageQuiz, quiz.Beginner)
	if err != nil {
		t.Fatalf("CreateQuestion() should fall back to random, got error = %v", err)
	}

	if question == nil {
		t.Fatal("CreateQuestion() returned nil")
	}
}
