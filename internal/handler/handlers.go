package handler

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/zalhui/URLShortener/internal/middleware"
)

type URLShortener struct {
	mu        sync.RWMutex
	urls      map[string]string
	shortened map[string]string
	filename  string
	baseURL   string
	nextUUID  int
}

type URLRecord struct {
	UUID        string `json:"uuid"`
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}

type ShortenURLRequest struct {
	URL string `json:"url"`
}

func NewURLShortener(baseURL string, filename string) *URLShortener {
	us := &URLShortener{
		urls:      make(map[string]string),
		shortened: make(map[string]string),
		baseURL:   baseURL,
		filename:  filename,
		nextUUID:  1,
	}
	if filename != "" {
		if err := us.loadURLsFromFile(filename); err != nil {
			fmt.Printf("failed to load URLs from file: %v", err)
		}
	}
	return us

}

func (us *URLShortener) loadURLsFromFile(filename string) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	maxUUID := 0
	for scanner.Scan() {
		var record URLRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return fmt.Errorf("failed to unmarshal record: %w", err)
		}
		uuid, err := strconv.Atoi(record.UUID)
		if err != nil {
			continue
		}
		if uuid > maxUUID {
			maxUUID = uuid
		}

		us.urls[record.ShortURL] = record.OriginalURL
		us.shortened[record.OriginalURL] = record.ShortURL
	}

	us.nextUUID = maxUUID + 1

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan file: %w", err)
	}

	return nil
}

func (us *URLShortener) saveURLsToFile(uuid, shortID, originalURL string) error {
	if us.filename == "" {
		return nil
	}

	us.mu.RLock()
	defer us.mu.RUnlock()

	file, err := os.OpenFile(us.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	record := URLRecord{
		UUID:        uuid,
		OriginalURL: originalURL,
		ShortURL:    fmt.Sprintf("%s%s", us.baseURL, shortID),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}
	if _, err = file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write record to file: %w", err)
	}

	return nil
}

func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate short ID: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (us *URLShortener) getNextUUID() string {
	uuid := us.nextUUID
	us.nextUUID++

	return strconv.Itoa(uuid)
}

func (us *URLShortener) shortenURL(originalURL string) (string, error) {
	us.mu.Lock()
	defer us.mu.Unlock()

	if shortID, exists := us.shortened[originalURL]; exists {
		return fmt.Sprintf("%s%s", us.baseURL, shortID), nil
	}

	shortID, err := generateShortID()
	if err != nil {
		return "", fmt.Errorf("failed to generate short ID: %w", err)
	}

	for {
		if _, exists := us.urls[shortID]; !exists {
			break
		}
		shortID, err = generateShortID()
		if err != nil {
			return "", err
		}
	}

	us.urls[shortID] = originalURL
	us.shortened[originalURL] = shortID

	uuid := us.getNextUUID()
	go func(uuid, shortID, originalURL string) {
		if err := us.saveURLsToFile(uuid, shortID, originalURL); err != nil {
			fmt.Printf("failed to save URLs to file: %v", err)
		}
	}(uuid, shortID, originalURL)

	return fmt.Sprintf("%s%s", us.baseURL, shortID), nil
}

func (us *URLShortener) getOriginalURL(shortID string) (string, bool) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	originalURL, exists := us.urls[shortID]
	return originalURL, exists
}

func (us *URLShortener) shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Unsupported Content-Type", http.StatusBadRequest)
		return
	}

	originalURL, err := io.ReadAll(r.Body)
	if err != nil || len(originalURL) == 0 {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	shortURL, err := us.shortenURL(string(originalURL))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (us *URLShortener) getOriginalURLHandler(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "shortID")

	originalURL, exists := us.getOriginalURL(shortID)
	if !exists {
		http.NotFound(w, r)
		return
	}

	//ensure that the original URL starts with http:// or https://
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		originalURL = "https://" + originalURL
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (us *URLShortener) JSONShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("content-type") != "application/json" {
		http.Error(w, "Unsupported content-type", http.StatusBadRequest)
		return
	}

	var req ShortenURLRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortURL, err := us.shortenURL(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"result": shortURL})

}

func (us *URLShortener) URLRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.LoggingMidlleware)
	r.Get("/{shortID}", us.getOriginalURLHandler)
	r.Post("/", us.shortenURLHandler)
	r.Post("/api/shorten", us.JSONShortenHandler)
	return r
}
