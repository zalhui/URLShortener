package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zalhui/URLShortener/internal/entity"
	"github.com/zalhui/URLShortener/internal/logger"
	"github.com/zalhui/URLShortener/internal/service"
)

func TestShortenURLHandler(t *testing.T) {
	logger.InitTest()

	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantCode    int
		wantBody    string
	}{
		{
			name:        "successful shorten",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://google.com",
			wantCode:    http.StatusCreated,
			wantBody:    "http://localhost:8080/",
		},
		{
			name:        "wrong method",
			method:      http.MethodGet,
			contentType: "text/plain",
			body:        "https://google.com",
			wantCode:    http.StatusMethodNotAllowed,
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        "https://google.com",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "url without scheme is normalized",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "google.com",
			wantCode:    http.StatusCreated,
			wantBody:    "http://localhost:8080/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestHandler().URLRouter()

			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if tt.wantBody != "" && !bytes.HasPrefix(w.Body.Bytes(), []byte(tt.wantBody)) {
				t.Fatalf("body = %q, want prefix %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestGetOriginalURLHandler(t *testing.T) {
	logger.InitTest()

	h := newTestHandler()
	shortened, err := h.service.ShortenURL(context.Background(), "", "https://google.com")
	if err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}

	tests := []struct {
		name         string
		method       string
		path         string
		wantCode     int
		wantLocation string
	}{
		{
			name:         "successful get",
			method:       http.MethodGet,
			path:         "/" + shortened.UUID,
			wantCode:     http.StatusTemporaryRedirect,
			wantLocation: "https://google.com",
		},
		{
			name:     "wrong method",
			method:   http.MethodPost,
			path:     "/" + shortened.UUID,
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			name:     "wrong id",
			method:   http.MethodGet,
			path:     "/missing",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := h.URLRouter()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if tt.wantLocation != "" && w.Header().Get("Location") != tt.wantLocation {
				t.Fatalf("location = %q, want %q", w.Header().Get("Location"), tt.wantLocation)
			}
		})
	}
}

func TestJSONShortenHandler(t *testing.T) {
	logger.InitTest()

	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantCode    int
	}{
		{
			name:        "successful shorten",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"https://google.com"}`,
			wantCode:    http.StatusCreated,
		},
		{
			name:        "wrong method",
			method:      http.MethodGet,
			contentType: "application/json",
			body:        `{"url":"https://google.com"}`,
			wantCode:    http.StatusMethodNotAllowed,
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://google.com",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        "",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "invalid url",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"url":"http://"}`,
			wantCode:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestHandler().URLRouter()
			req := httptest.NewRequest(tt.method, "/api/shorten", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if w.Code == http.StatusCreated {
				var resp map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("response is not valid json: %v", err)
				}
				if resp["result"] == "" {
					t.Fatal("result is empty")
				}
			}
		})
	}
}

func newTestHandler() *ShortenHandler {
	repo := newStubURLRepository()
	svc := service.NewShortenerService(repo, "http://localhost:8080", nil)
	return NewShortenHandler(svc)
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
	recordCopy := *url
	s.urls[url.UUID] = &recordCopy
	s.shortened[url.UserID+"|"+url.OriginalURL] = &recordCopy
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
	url, exists := s.shortened[userID+"|"+originalURL]
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
	delete(s.shortened, userID+"|"+url.OriginalURL)
	return nil
}

func (s *stubURLRepository) Close() error {
	return nil
}
