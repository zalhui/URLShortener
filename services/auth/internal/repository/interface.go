package repository

import (
	"context"
	"errors"

	"github.com/zalhui/URLShortener/auth/internal/entity"
)

var (
	ErrEmailTaken      = errors.New("email already taken")
	ErrSessionNotFound = errors.New("session not found")
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	GetUserByID(ctx context.Context, id string) (*entity.User, error)
	CreateSession(ctx context.Context, session *entity.Session) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*entity.Session, error)
	RotateSession(ctx context.Context, currentTokenHash string, newSession *entity.Session) error
	RevokeSession(ctx context.Context, tokenHash string) error
	Close() error
}
