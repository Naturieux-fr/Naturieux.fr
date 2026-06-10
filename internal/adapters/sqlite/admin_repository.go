package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// Credentials returns the authentication data for a username.
func (r *PlayerRepository) Credentials(ctx context.Context, username string) (ports.Account, error) {
	var c ports.Account
	err := r.db.QueryRowContext(ctx,
		`SELECT id, password_hash, role FROM players WHERE username = ?`, username).
		Scan(&c.ID, &c.PasswordHash, &c.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.Account{}, ports.ErrNotFound
	}
	if err != nil {
		return ports.Account{}, fmt.Errorf("loading credentials: %w", err)
	}
	return c, nil
}

// Role returns a player's role, or ErrNotFound if the player is unknown.
func (r *PlayerRepository) Role(ctx context.Context, id string) (string, error) {
	var role string
	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM players WHERE id = ?`, id).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ports.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("loading role: %w", err)
	}
	return role, nil
}

// UpsertAdmin creates or updates an admin account with the given username,
// password hash and id. The id is only used when creating the account.
func (r *PlayerRepository) UpsertAdmin(ctx context.Context, id, username, passwordHash, createdAt string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO players (id, username, created_at, role, password_hash)
		VALUES (?, ?, ?, 'admin', ?)
		ON CONFLICT(username) DO UPDATE SET
			role = 'admin',
			password_hash = excluded.password_hash`,
		id, username, createdAt, passwordHash)
	if err != nil {
		return fmt.Errorf("upserting admin: %w", err)
	}
	return nil
}

// Ensure interface compliance.
var _ ports.AccountStore = (*PlayerRepository)(nil)
