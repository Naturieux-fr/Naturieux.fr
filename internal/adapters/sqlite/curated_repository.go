package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// CuratedRepository stores admin-composed quizzes and their scheduling as the
// daily/weekly challenge.
type CuratedRepository struct {
	db *sql.DB
}

// NewCuratedRepository creates a curated-quiz repository.
func NewCuratedRepository(db *sql.DB) *CuratedRepository {
	return &CuratedRepository{db: db}
}

// CuratedQuiz is an admin-composed selection of species.
type CuratedQuiz struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Species   []int  `json:"species"`
	CreatedAt string `json:"created_at"`
}

// CreateQuiz stores a curated quiz.
func (r *CuratedRepository) CreateQuiz(ctx context.Context, id, name string, species []int, createdAt string) error {
	raw, err := json.Marshal(species)
	if err != nil {
		return fmt.Errorf("encoding species: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO curated_quizzes (id, name, species, created_at) VALUES (?, ?, ?, ?)`,
		id, name, string(raw), createdAt)
	if err != nil {
		return fmt.Errorf("creating quiz: %w", err)
	}
	return nil
}

// ListQuizzes returns all curated quizzes, newest first.
func (r *CuratedRepository) ListQuizzes(ctx context.Context) ([]CuratedQuiz, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, species, created_at FROM curated_quizzes ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing quizzes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]CuratedQuiz, 0)
	for rows.Next() {
		q, raw := CuratedQuiz{}, ""
		if err := rows.Scan(&q.ID, &q.Name, &raw, &q.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning quiz: %w", err)
		}
		_ = json.Unmarshal([]byte(raw), &q.Species)
		out = append(out, q)
	}
	return out, rows.Err()
}

// QuizSpecies returns the species ids of a curated quiz.
func (r *CuratedRepository) QuizSpecies(ctx context.Context, id string) ([]int, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT species FROM curated_quizzes WHERE id = ?`, id).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("loading quiz: %w", err)
	}
	var species []int
	_ = json.Unmarshal([]byte(raw), &species)
	return species, nil
}

// DeleteQuiz removes a curated quiz and any schedule pointing at it.
func (r *CuratedRepository) DeleteQuiz(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM curated_quizzes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting quiz: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ports.ErrNotFound
	}
	_, _ = r.db.ExecContext(ctx, `DELETE FROM challenge_schedule WHERE quiz_id = ?`, id)
	return nil
}

// Schedule sets (or replaces) the curated quiz used as the challenge for a
// period key.
func (r *CuratedRepository) Schedule(ctx context.Context, period, key, quizID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO challenge_schedule (period, period_key, quiz_id) VALUES (?, ?, ?)
		ON CONFLICT(period, period_key) DO UPDATE SET quiz_id = excluded.quiz_id`,
		period, key, quizID)
	if err != nil {
		return fmt.Errorf("scheduling challenge: %w", err)
	}
	return nil
}

// Scheduled returns the curated quiz id scheduled for a period key, if any.
func (r *CuratedRepository) Scheduled(ctx context.Context, period, key string) (string, bool, error) {
	var id string
	err := r.db.QueryRowContext(ctx,
		`SELECT quiz_id FROM challenge_schedule WHERE period = ? AND period_key = ?`, period, key).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("loading schedule: %w", err)
	}
	return id, true, nil
}
