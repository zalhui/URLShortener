package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/zalhui/URLShortener/internal/entity"
)

type FileRepository struct {
	mu        sync.RWMutex
	fileMu    sync.Mutex
	filename  string
	urls      map[string]*entity.URL
	shortened map[string]*entity.URL
}

func NewFileRepository(filename string) (*FileRepository, error) {
	repo := &FileRepository{
		filename:  filename,
		urls:      make(map[string]*entity.URL),
		shortened: make(map[string]*entity.URL),
	}

	if err := repo.LoadFromFile(filename); err != nil {
		return nil, err
	}

	return repo, nil
}

func (f *FileRepository) Save(_ context.Context, url *entity.URL) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.urls[url.UUID]; exists {
		return ErrDuplicateShortID
	}
	if _, exists := f.shortened[userScopedKey(url.UserID, url.OriginalURL)]; exists {
		return ErrDuplicateOriginalURL
	}

	recordCopy := *url
	f.urls[url.UUID] = &recordCopy
	f.shortened[userScopedKey(url.UserID, url.OriginalURL)] = &recordCopy

	return f.saveToFile(&recordCopy)
}

func (f *FileRepository) GetByShortID(_ context.Context, shortID string) (*entity.URL, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	url, exists := f.urls[shortID]
	if !exists {
		return nil, nil
	}

	recordCopy := *url
	return &recordCopy, nil
}

func (f *FileRepository) GetByOriginalURL(_ context.Context, userID, originalURL string) (*entity.URL, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	url, exists := f.shortened[userScopedKey(userID, originalURL)]
	if !exists {
		return nil, nil
	}

	recordCopy := *url
	return &recordCopy, nil
}

func (f *FileRepository) ListByUser(_ context.Context, userID string) ([]*entity.URL, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	urls := make([]*entity.URL, 0)
	for _, url := range f.urls {
		if url.UserID != userID {
			continue
		}

		recordCopy := *url
		urls = append(urls, &recordCopy)
	}

	return urls, nil
}

func (f *FileRepository) DeleteByShortID(_ context.Context, userID, shortID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	url, exists := f.urls[shortID]
	if !exists || url.UserID != userID {
		return nil
	}

	delete(f.urls, shortID)
	delete(f.shortened, userScopedKey(userID, url.OriginalURL))

	return f.rewriteFile()
}

func (f *FileRepository) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record entity.URL
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return fmt.Errorf("failed to unmarshal record: %w", err)
		}

		recordCopy := record
		f.urls[record.UUID] = &recordCopy
		f.shortened[userScopedKey(record.UserID, record.OriginalURL)] = &recordCopy
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan file: %w", err)
	}

	return nil
}

func (f *FileRepository) saveToFile(url *entity.URL) error {
	if f.filename == "" {
		return nil
	}

	f.fileMu.Lock()
	defer f.fileMu.Unlock()

	file, err := os.OpenFile(f.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}
	if _, err = file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write record to file: %w", err)
	}

	return nil
}

func (f *FileRepository) rewriteFile() error {
	if f.filename == "" {
		return nil
	}

	f.fileMu.Lock()
	defer f.fileMu.Unlock()

	file, err := os.OpenFile(f.filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file for rewrite: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, url := range f.urls {
		if err := encoder.Encode(url); err != nil {
			return fmt.Errorf("failed to rewrite record: %w", err)
		}
	}

	return nil
}

func (f *FileRepository) Close() error {
	return nil
}

func userScopedKey(userID, originalURL string) string {
	return userID + "|" + originalURL
}
