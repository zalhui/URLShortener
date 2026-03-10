package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zalhui/URLShortener/auth/internal/entity"
	"github.com/zalhui/URLShortener/auth/internal/repository"
	"github.com/zalhui/URLShortener/auth/internal/token"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 12

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrWeakPassword       = errors.New("password does not meet security requirements")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

type RequestMeta struct {
	UserAgent string
	IPAddress string
}

type AuthResult struct {
	AccessToken          string
	AccessTokenExpiresAt time.Time
	RefreshToken         string
	User                 *entity.User
}

type AccessIdentity struct {
	UserID string
	Email  string
}

type TokenManager interface {
	CreateAccessToken(user *entity.User) (string, time.Time, error)
	ParseAccessToken(tokenString string) (*token.AccessClaims, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type AuthService struct {
	repo            repository.AuthRepository
	tokenManager    TokenManager
	pinger          Pinger
	refreshTokenTTL time.Duration
}

func NewAuthService(repo repository.AuthRepository, tokenManager TokenManager, pinger Pinger, refreshTokenTTL time.Duration) *AuthService {
	return &AuthService{
		repo:            repo,
		tokenManager:    tokenManager,
		pinger:          pinger,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *AuthService) Ping(ctx context.Context) error {
	if s.pinger == nil {
		return nil
	}

	return s.pinger.Ping(ctx)
}

func (s *AuthService) Register(ctx context.Context, email, password string, meta RequestMeta) (*AuthResult, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &entity.User{
		ID:           uuid.NewString(),
		Email:        normalizedEmail,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	return s.issueTokens(ctx, user, meta)
}

func (s *AuthService) Login(ctx context.Context, email, password string, meta RequestMeta) (*AuthResult, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user, meta)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string, meta RequestMeta) (*AuthResult, error) {
	if rawRefreshToken == "" {
		return nil, ErrUnauthorized
	}

	tokenHash := token.HashRefreshToken(rawRefreshToken)
	session, err := s.repo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if session == nil || session.RevokedAt != nil || session.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUnauthorized
	}

	refreshToken, refreshHash, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	newSession := &entity.Session{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: refreshHash,
		UserAgent: meta.UserAgent,
		IPAddress: meta.IPAddress,
		ExpiresAt: time.Now().UTC().Add(s.refreshTokenTTL),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.RotateSession(ctx, tokenHash, newSession); err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	accessToken, accessTokenExpiresAt, err := s.tokenManager.CreateAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
		RefreshToken:         refreshToken,
		User:                 user,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}

	return s.repo.RevokeSession(ctx, token.HashRefreshToken(rawRefreshToken))
}

func (s *AuthService) CurrentUser(ctx context.Context, accessToken string) (*entity.User, error) {
	identity, err := s.ParseAccessToken(accessToken)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}

func (s *AuthService) ParseAccessToken(accessToken string) (*AccessIdentity, error) {
	if accessToken == "" {
		return nil, ErrUnauthorized
	}

	claims, err := s.tokenManager.ParseAccessToken(accessToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if claims.Subject == "" {
		return nil, ErrUnauthorized
	}

	return &AccessIdentity{
		UserID: claims.Subject,
		Email:  claims.Email,
	}, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *entity.User, meta RequestMeta) (*AuthResult, error) {
	refreshToken, refreshHash, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	session := &entity.Session{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: refreshHash,
		UserAgent: meta.UserAgent,
		IPAddress: meta.IPAddress,
		ExpiresAt: time.Now().UTC().Add(s.refreshTokenTTL),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	accessToken, accessTokenExpiresAt, err := s.tokenManager.CreateAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
		RefreshToken:         refreshToken,
		User:                 user,
	}, nil
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(normalized)
	if err != nil {
		return "", ErrInvalidEmail
	}
	if address.Address == "" {
		return "", ErrInvalidEmail
	}

	return address.Address, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrWeakPassword
	}

	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}
