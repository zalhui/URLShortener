package repository

import (
	"sync"

	"github.com/zalhui/URLShortener/internal/entity"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	urls      map[string]*entity.URL
	shortened map[string]*entity.URL
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls:      make(map[string]*entity.URL),
		shortened: make(map[string]*entity.URL),
	}
}

func (m *MemoryRepository) Save(url *entity.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.urls[url.UUID] = url
	m.shortened[url.OriginalURL] = url
	return nil
}

func (m *MemoryRepository) GetByShortID(shortID string) (*entity.URL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	originalURL, exists := m.urls[shortID]
	if !exists {
		return nil, nil
	}
	return originalURL, nil
}

func (m *MemoryRepository) GetByOriginalURL(originalURL string) (*entity.URL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shortID, exists := m.shortened[originalURL]
	if !exists {
		return nil, nil
	}
	return shortID, nil
}

func (m *MemoryRepository) Close() error {
	return nil
}
