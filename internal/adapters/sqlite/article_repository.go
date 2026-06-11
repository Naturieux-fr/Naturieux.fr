package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// ArticleRepository stores learning articles written by rédacteurs.
type ArticleRepository struct {
	db *sql.DB
}

// NewArticleRepository creates an article repository.
func NewArticleRepository(db *sql.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

// ArticleSpecies links an article to a species (for quiz redirection).
type ArticleSpecies struct {
	CdNom int    `json:"cd_nom"`
	Name  string `json:"name"`
}

// Article is a learning article.
type Article struct {
	ID         string           `json:"id"`
	Title      string           `json:"title"`
	Body       string           `json:"body"`
	Species    []ArticleSpecies `json:"species"`
	AuthorID   string           `json:"author_id"`
	AuthorName string           `json:"author_name"`
	Published  bool             `json:"published"`
	CreatedAt  string           `json:"created_at"`
	UpdatedAt  string           `json:"updated_at"`
}

// Save inserts or updates an article (upsert by id).
func (r *ArticleRepository) Save(ctx context.Context, a Article) error {
	species, err := json.Marshal(a.Species)
	if err != nil {
		return fmt.Errorf("encoding species: %w", err)
	}
	pub := 0
	if a.Published {
		pub = 1
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO articles (id, title, body, species, author_id, author_name, published, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title, body=excluded.body, species=excluded.species,
			published=excluded.published, updated_at=excluded.updated_at`,
		a.ID, a.Title, a.Body, string(species), a.AuthorID, a.AuthorName, pub, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("saving article: %w", err)
	}
	return nil
}

// Get returns one article by id.
func (r *ArticleRepository) Get(ctx context.Context, id string) (Article, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, title, body, species, author_id, author_name, published, created_at, updated_at FROM articles WHERE id = ?`, id)
	return scanArticle(row)
}

// Delete removes an article.
func (r *ArticleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting article: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// List returns articles, optionally only published ones, newest first.
func (r *ArticleRepository) List(ctx context.Context, publishedOnly bool) ([]Article, error) {
	q := `SELECT id, title, body, species, author_id, author_name, published, created_at, updated_at FROM articles`
	if publishedOnly {
		q += ` WHERE published = 1`
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("listing articles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]Article, 0)
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// BySpecies returns published articles that cover the given cd_nom.
func (r *ArticleRepository) BySpecies(ctx context.Context, cdNom int) ([]Article, error) {
	all, err := r.List(ctx, true)
	if err != nil {
		return nil, err
	}
	out := make([]Article, 0)
	for _, a := range all {
		for _, s := range a.Species {
			if s.CdNom == cdNom {
				out = append(out, a)
				break
			}
		}
	}
	return out, nil
}

type rowScannerArt interface {
	Scan(dest ...interface{}) error
}

func scanArticle(row rowScannerArt) (Article, error) {
	var a Article
	var species string
	var pub int
	err := row.Scan(&a.ID, &a.Title, &a.Body, &species, &a.AuthorID, &a.AuthorName, &pub, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return Article{}, ports.ErrNotFound
	}
	if err != nil {
		return Article{}, fmt.Errorf("scanning article: %w", err)
	}
	a.Published = pub != 0
	_ = json.Unmarshal([]byte(species), &a.Species)
	return a, nil
}
