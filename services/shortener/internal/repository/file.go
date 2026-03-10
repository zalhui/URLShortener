package repository

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/zalhui/URLShortener/internal/entity"
)

type FileRepository struct {
	*MemoryRepository
	fileMu   sync.Mutex
	filename string
}

func NewFileRepository(filename string) (*FileRepository, error) {
	repo := &FileRepository{
		MemoryRepository: NewMemoryRepository(),
		filename:         filename,
	}

	if err := repo.LoadFromFile(filename); err != nil {
		return nil, err
	}
	return repo, nil
}

func (f *FileRepository) Save(url *entity.URL) error {
	if err := f.MemoryRepository.Save(url); err != nil {
		return err
	}

	return f.SaveToFile(url)
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
		f.shortened[record.OriginalURL] = &recordCopy
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan file: %w", err)
	}

	return nil
}

func (f *FileRepository) SaveToFile(url *entity.URL) error {
	if f.filename == "" {
		return nil
	}

	f.fileMu.Lock()
	defer f.fileMu.Unlock()

	file, err := os.OpenFile(f.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

func (f *FileRepository) Close() error {
	return nil
}
