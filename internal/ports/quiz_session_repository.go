package ports

import (
	"context"

	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz"
)

// QuizSessionRepository defines the interface for quiz session persistence.
type QuizSessionRepository interface {
	// Save persists a quiz session.
	Save(ctx context.Context, session *quiz.Session) error

	// GetByID retrieves a session by ID.
	GetByID(ctx context.Context, id string) (*quiz.Session, error)

	// GetByUserID retrieves sessions for a user.
	GetByUserID(ctx context.Context, userID string, limit int) ([]*quiz.Session, error)

	// GetStats retrieves aggregated stats for a user.
	GetStats(ctx context.Context, userID string) (*UserQuizStats, error)
}

// UserQuizStats contains aggregated quiz statistics.
type UserQuizStats struct {
	TotalSessions   int
	TotalQuestions  int
	TotalCorrect    int
	TotalScore      int
	AverageAccuracy float64
	BestStreak      int
	FavoriteTaxon   string
}
