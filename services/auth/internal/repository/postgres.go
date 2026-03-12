package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zalhui/URLShortener/auth/internal/database"
	"github.com/zalhui/URLShortener/auth/internal/entity"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *entity.User) error {
	const query = `
INSERT INTO auth_users (id, email, password_hash, created_at)
VALUES ($1, $2, $3, $4)
`

	if _, err := r.db.Pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "auth_users_email_key" {
			return ErrEmailTaken
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	const query = `
SELECT id, email, password_hash, created_at
FROM auth_users
WHERE email = $1
`

	user := &entity.User{}
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	const query = `
SELECT id, email, password_hash, created_at
FROM auth_users
WHERE id = $1
`

	user := &entity.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session *entity.Session) error {
	const query = `
INSERT INTO auth_sessions (
    id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, revoked_at, replaced_by_hash
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

	if _, err := r.db.Pool.Exec(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
		session.CreatedAt,
		session.RevokedAt,
		session.ReplacedByHash,
	); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*entity.Session, error) {
	const query = `
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, revoked_at, replaced_by_hash
FROM auth_sessions
WHERE token_hash = $1
`

	session := &entity.Session{}
	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.RevokedAt,
		&session.ReplacedByHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session by token hash: %w", err)
	}

	return session, nil
}

func (r *PostgresRepository) RotateSession(ctx context.Context, currentTokenHash string, newSession *entity.Session) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start rotation transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
UPDATE auth_sessions
SET revoked_at = NOW(), replaced_by_hash = $2
WHERE token_hash = $1 AND revoked_at IS NULL
`, currentTokenHash, newSession.TokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke current session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO auth_sessions (
    id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, revoked_at, replaced_by_hash
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`,
		newSession.ID,
		newSession.UserID,
		newSession.TokenHash,
		newSession.UserAgent,
		newSession.IPAddress,
		newSession.ExpiresAt,
		newSession.CreatedAt,
		newSession.RevokedAt,
		newSession.ReplacedByHash,
	); err != nil {
		return fmt.Errorf("failed to create rotated session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rotation transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, tokenHash string) error {
	if tokenHash == "" {
		return nil
	}

	if _, err := r.db.Pool.Exec(ctx, `
UPDATE auth_sessions
SET revoked_at = COALESCE(revoked_at, NOW())
WHERE token_hash = $1
`, tokenHash); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) Close() error {
	return nil
}
