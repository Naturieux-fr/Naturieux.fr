package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Naturieux-fr/Naturieux.fr/internal/ports"
)

// InviteRepository persists registration invitations.
type InviteRepository struct {
	db *sql.DB
}

// NewInviteRepository creates an invite repository.
func NewInviteRepository(db *sql.DB) *InviteRepository {
	return &InviteRepository{db: db}
}

// CreateInvite stores a new pending invitation.
func (r *InviteRepository) CreateInvite(ctx context.Context, token, createdBy, createdAt string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO invites (token, created_by, created_at) VALUES (?, ?, ?)`,
		token, createdBy, createdAt)
	if err != nil {
		return fmt.Errorf("creating invite: %w", err)
	}
	return nil
}

// GetInvite returns an invitation by token.
func (r *InviteRepository) GetInvite(ctx context.Context, token string) (ports.Invite, error) {
	var inv ports.Invite
	var usedBy, usedAt sql.NullString
	var revoked int
	err := r.db.QueryRowContext(ctx,
		`SELECT token, created_by, created_at, used_by, used_at, revoked FROM invites WHERE token = ?`, token).
		Scan(&inv.Token, &inv.CreatedBy, &inv.CreatedAt, &usedBy, &usedAt, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.Invite{}, ports.ErrNotFound
	}
	if err != nil {
		return ports.Invite{}, fmt.Errorf("loading invite: %w", err)
	}
	inv.UsedBy, inv.UsedAt, inv.Revoked = usedBy.String, usedAt.String, revoked != 0
	return inv, nil
}

// ConsumeInvite atomically marks an invite used, but only when it is still
// pending (unused, not revoked) and not older than minCreatedAt. It returns
// true only if exactly one row was consumed, making invites strictly
// single-use even under concurrent registrations.
func (r *InviteRepository) ConsumeInvite(ctx context.Context, token, usedBy, usedAt, minCreatedAt string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invites SET used_by = ?, used_at = ?
		WHERE token = ? AND used_by IS NULL AND revoked = 0 AND created_at >= ?`,
		usedBy, usedAt, token, minCreatedAt)
	if err != nil {
		return false, fmt.Errorf("consuming invite: %w", err)
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// RevokeInvite disables an invitation.
func (r *InviteRepository) RevokeInvite(ctx context.Context, token string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE invites SET revoked = 1 WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("revoking invite: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ListInvites returns invitations, newest first.
func (r *InviteRepository) ListInvites(ctx context.Context, limit int) ([]ports.Invite, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT token, created_by, created_at, used_by, used_at, revoked FROM invites ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing invites: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]ports.Invite, 0, limit)
	for rows.Next() {
		var inv ports.Invite
		var usedBy, usedAt sql.NullString
		var revoked int
		if err := rows.Scan(&inv.Token, &inv.CreatedBy, &inv.CreatedAt, &usedBy, &usedAt, &revoked); err != nil {
			return nil, fmt.Errorf("scanning invite: %w", err)
		}
		inv.UsedBy, inv.UsedAt, inv.Revoked = usedBy.String, usedAt.String, revoked != 0
		out = append(out, inv)
	}
	return out, rows.Err()
}

var _ ports.InviteStore = (*InviteRepository)(nil)
