package service

import (
	"context"
	"testing"
	"time"

	"github.com/zalhui/URLShortener/auth/internal/entity"
	"github.com/zalhui/URLShortener/auth/internal/logger"
	"github.com/zalhui/URLShortener/auth/internal/repository"
	"github.com/zalhui/URLShortener/auth/internal/token"
)

func TestRegisterLoginRefreshFlow(t *testing.T) {
	logger.InitTest()

	repo := newStubAuthRepository()
	tokenManager, err := token.NewManager("supersecretkeysupersecretkey123456", "auth-service", 15*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	svc := NewAuthService(repo, tokenManager, nil, time.Hour)

	registerResult, err := svc.Register(context.Background(), "User@example.com", "very-secure-password", RequestMeta{
		UserAgent: "test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registerResult.User.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", registerResult.User.Email)
	}
	if registerResult.RefreshToken == "" || registerResult.AccessToken == "" {
		t.Fatal("expected issued tokens")
	}

	loginResult, err := svc.Login(context.Background(), "user@example.com", "very-secure-password", RequestMeta{})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loginResult.User.ID != registerResult.User.ID {
		t.Fatalf("expected same user id, got %q and %q", loginResult.User.ID, registerResult.User.ID)
	}

	refreshResult, err := svc.Refresh(context.Background(), loginResult.RefreshToken, RequestMeta{})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshResult.RefreshToken == loginResult.RefreshToken {
		t.Fatal("expected rotated refresh token")
	}

	currentUser, err := svc.CurrentUser(context.Background(), refreshResult.AccessToken)
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if currentUser.ID != registerResult.User.ID {
		t.Fatalf("expected authenticated user id %q, got %q", registerResult.User.ID, currentUser.ID)
	}

	if err := svc.Logout(context.Background(), refreshResult.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := svc.Refresh(context.Background(), refreshResult.RefreshToken, RequestMeta{}); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized after logout, got %v", err)
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	repo := newStubAuthRepository()
	tokenManager, err := token.NewManager("supersecretkeysupersecretkey123456", "auth-service", 15*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	svc := NewAuthService(repo, tokenManager, nil, time.Hour)
	if _, err := svc.Register(context.Background(), "user@example.com", "short", RequestMeta{}); err != ErrWeakPassword {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
}

type stubAuthRepository struct {
	usersByID    map[string]*entity.User
	usersByEmail map[string]*entity.User
	sessions     map[string]*entity.Session
}

func newStubAuthRepository() *stubAuthRepository {
	return &stubAuthRepository{
		usersByID:    make(map[string]*entity.User),
		usersByEmail: make(map[string]*entity.User),
		sessions:     make(map[string]*entity.Session),
	}
}

func (s *stubAuthRepository) CreateUser(_ context.Context, user *entity.User) error {
	if _, exists := s.usersByEmail[user.Email]; exists {
		return repository.ErrEmailTaken
	}

	copyUser := *user
	s.usersByID[user.ID] = &copyUser
	s.usersByEmail[user.Email] = &copyUser
	return nil
}

func (s *stubAuthRepository) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	user, exists := s.usersByEmail[email]
	if !exists {
		return nil, nil
	}

	copyUser := *user
	return &copyUser, nil
}

func (s *stubAuthRepository) GetUserByID(_ context.Context, id string) (*entity.User, error) {
	user, exists := s.usersByID[id]
	if !exists {
		return nil, nil
	}

	copyUser := *user
	return &copyUser, nil
}

func (s *stubAuthRepository) CreateSession(_ context.Context, session *entity.Session) error {
	copySession := *session
	s.sessions[session.TokenHash] = &copySession
	return nil
}

func (s *stubAuthRepository) GetSessionByTokenHash(_ context.Context, tokenHash string) (*entity.Session, error) {
	session, exists := s.sessions[tokenHash]
	if !exists {
		return nil, nil
	}

	copySession := *session
	return &copySession, nil
}

func (s *stubAuthRepository) RotateSession(_ context.Context, currentTokenHash string, newSession *entity.Session) error {
	current, exists := s.sessions[currentTokenHash]
	if !exists || current.RevokedAt != nil {
		return repository.ErrSessionNotFound
	}

	now := time.Now().UTC()
	replacedBy := newSession.TokenHash
	current.RevokedAt = &now
	current.ReplacedByHash = &replacedBy

	copySession := *newSession
	s.sessions[newSession.TokenHash] = &copySession
	return nil
}

func (s *stubAuthRepository) RevokeSession(_ context.Context, tokenHash string) error {
	session, exists := s.sessions[tokenHash]
	if !exists || session.RevokedAt != nil {
		return nil
	}

	now := time.Now().UTC()
	session.RevokedAt = &now
	return nil
}

func (s *stubAuthRepository) Close() error {
	return nil
}
