package service

import (
	"context"
	"testing"

	"github.com/zalhui/URLShortener/internal/entity"
	"github.com/zalhui/URLShortener/internal/repository"
)

func TestShortenURLScopesDuplicatesByUser(t *testing.T) {
	repo := newStubURLRepository()
	svc := NewShortenerService(repo, "http://localhost:8080", nil)

	first, err := svc.ShortenURL(context.Background(), "user-1", "https://example.com")
	if err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}

	sameUser, err := svc.ShortenURL(context.Background(), "user-1", "https://example.com")
	if err != nil {
		t.Fatalf("ShortenURL() same user error = %v", err)
	}
	if first.UUID != sameUser.UUID {
		t.Fatalf("expected same short url for same user, got %q and %q", first.UUID, sameUser.UUID)
	}

	otherUser, err := svc.ShortenURL(context.Background(), "user-2", "https://example.com")
	if err != nil {
		t.Fatalf("ShortenURL() other user error = %v", err)
	}
	if first.UUID == otherUser.UUID {
		t.Fatalf("expected different short url for another user, got same %q", first.UUID)
	}
	if otherUser.UserID != "user-2" {
		t.Fatalf("expected user ownership to be stored, got %q", otherUser.UserID)
	}
}

type stubURLRepository struct {
	urls      map[string]*entity.URL
	shortened map[string]*entity.URL
}

func newStubURLRepository() *stubURLRepository {
	return &stubURLRepository{
		urls:      make(map[string]*entity.URL),
		shortened: make(map[string]*entity.URL),
	}
}

func (s *stubURLRepository) Save(_ context.Context, url *entity.URL) error {
	if _, exists := s.urls[url.UUID]; exists {
		return repository.ErrDuplicateShortID
	}
	if _, exists := s.shortened[testUserScopedKey(url.UserID, url.OriginalURL)]; exists {
		return repository.ErrDuplicateOriginalURL
	}

	recordCopy := *url
	s.urls[url.UUID] = &recordCopy
	s.shortened[testUserScopedKey(url.UserID, url.OriginalURL)] = &recordCopy
	return nil
}

func (s *stubURLRepository) GetByShortID(_ context.Context, shortID string) (*entity.URL, error) {
	url, exists := s.urls[shortID]
	if !exists {
		return nil, nil
	}

	recordCopy := *url
	return &recordCopy, nil
}

func (s *stubURLRepository) GetByOriginalURL(_ context.Context, userID, originalURL string) (*entity.URL, error) {
	url, exists := s.shortened[testUserScopedKey(userID, originalURL)]
	if !exists {
		return nil, nil
	}

	recordCopy := *url
	return &recordCopy, nil
}

func (s *stubURLRepository) ListByUser(_ context.Context, userID string) ([]*entity.URL, error) {
	urls := make([]*entity.URL, 0)
	for _, url := range s.urls {
		if url.UserID != userID {
			continue
		}
		recordCopy := *url
		urls = append(urls, &recordCopy)
	}
	return urls, nil
}

func (s *stubURLRepository) DeleteByShortID(_ context.Context, userID, shortID string) error {
	url, exists := s.urls[shortID]
	if !exists || url.UserID != userID {
		return nil
	}

	delete(s.urls, shortID)
	delete(s.shortened, testUserScopedKey(userID, url.OriginalURL))
	return nil
}

func (s *stubURLRepository) Close() error {
	return nil
}

func testUserScopedKey(userID, originalURL string) string {
	return userID + "|" + originalURL
}
