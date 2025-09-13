package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type URLShortener struct {
	mu        sync.RWMutex
	urls      map[string]string
	shortened map[string]string
}

func NewURLShortener() *URLShortener {
	return &URLShortener{
		urls:      make(map[string]string),
		shortened: make(map[string]string),
	}
}

func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate short ID: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (us *URLShortener) shortenURL(originalURL string) (string, error) {
	us.mu.Lock()
	defer us.mu.Unlock()

	if shortID, exists := us.shortened[originalURL]; exists {
		return fmt.Sprintf("http://localhost:8080/%s", shortID), nil
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

	return fmt.Sprintf("http://localhost:8080/%s", shortID), nil
}

func (us *URLShortener) getOriginalURL(shortID string) (string, bool) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	originalURL, exists := us.urls[shortID]
	return originalURL, exists
}

func (us *URLShortener) shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

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
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	shortID := r.URL.Path[1:]

	originalURL, exists := us.getOriginalURL(shortID)
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (us *URLShortener) rootHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		us.shortenURLHandler(w, r)
	case http.MethodGet:
		us.getOriginalURLHandler(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (us *URLShortener) StartServer() error {
	http.HandleFunc("/", us.rootHandler)

	log.Println("Server started on :8080")
	return http.ListenAndServe(":8080", nil)

}
