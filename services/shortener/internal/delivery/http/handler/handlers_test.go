package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	shortenerauth "github.com/zalhui/URLShortener/internal/auth"
	"github.com/zalhui/URLShortener/internal/delivery/http/dto"
	transportmiddleware "github.com/zalhui/URLShortener/internal/delivery/http/middleware"
	"github.com/zalhui/URLShortener/internal/entity"
	"github.com/zalhui/URLShortener/internal/logger"
	"github.com/zalhui/URLShortener/internal/service"
)

const (
	testAccessTokenSecret = "test-access-token-secret-must-be-at-least-32-chars"
	testAccessTokenIssuer = "auth-service"
)

func TestShortenURLHandler(t *testing.T) {
	logger.InitTest()

	tests := []struct {
		name          string
		method        string
		contentType   string
		body          string
		tokenUserID   string
		wantCode      int
		wantBody      string
		authorization bool
	}{
		{
			name:          "successful shorten",
			method:        http.MethodPost,
			contentType:   "text/plain",
			body:          "https://google.com",
			tokenUserID:   "user-1",
			wantCode:      http.StatusCreated,
			wantBody:      "http://localhost:8080/",
			authorization: true,
		},
		{
			name:          "unauthorized without token",
			method:        http.MethodPost,
			contentType:   "text/plain",
			body:          "https://google.com",
			wantCode:      http.StatusUnauthorized,
			authorization: false,
		},
		{
			name:          "wrong method",
			method:        http.MethodGet,
			contentType:   "text/plain",
			body:          "https://google.com",
			wantCode:      http.StatusMethodNotAllowed,
			authorization: true,
			tokenUserID:   "user-1",
		},
		{
			name:          "wrong content type",
			method:        http.MethodPost,
			contentType:   "application/json",
			body:          "https://google.com",
			wantCode:      http.StatusBadRequest,
			authorization: true,
			tokenUserID:   "user-1",
		},
		{
			name:          "empty body",
			method:        http.MethodPost,
			contentType:   "text/plain",
			body:          "",
			wantCode:      http.StatusBadRequest,
			authorization: true,
			tokenUserID:   "user-1",
		},
		{
			name:          "url without scheme is normalized",
			method:        http.MethodPost,
			contentType:   "text/plain",
			body:          "google.com",
			wantCode:      http.StatusCreated,
			wantBody:      "http://localhost:8080/",
			authorization: true,
			tokenUserID:   "user-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestHandler(t).URLRouter()

			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			if tt.authorization {
				req.Header.Set("Authorization", "Bearer "+newAccessToken(t, tt.tokenUserID))
			}
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

	h := newTestHandler(t)
	shortened, err := h.service.ShortenURL(context.Background(), "user-1", "https://google.com")
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
		name          string
		method        string
		contentType   string
		body          string
		tokenUserID   string
		wantCode      int
		authorization bool
	}{
		{
			name:          "successful shorten",
			method:        http.MethodPost,
			contentType:   "application/json",
			body:          `{"url":"https://google.com"}`,
			tokenUserID:   "user-1",
			wantCode:      http.StatusCreated,
			authorization: true,
		},
		{
			name:          "unauthorized without token",
			method:        http.MethodPost,
			contentType:   "application/json",
			body:          `{"url":"https://google.com"}`,
			wantCode:      http.StatusUnauthorized,
			authorization: false,
		},
		{
			name:          "wrong method",
			method:        http.MethodGet,
			contentType:   "application/json",
			body:          `{"url":"https://google.com"}`,
			wantCode:      http.StatusMethodNotAllowed,
			authorization: true,
			tokenUserID:   "user-1",
		},
		{
			name:          "wrong content type",
			method:        http.MethodPost,
			contentType:   "text/plain",
			body:          "https://google.com",
			wantCode:      http.StatusBadRequest,
			authorization: true,
			tokenUserID:   "user-1",
		},
		{
			name:          "empty body",
			method:        http.MethodPost,
			contentType:   "application/json",
			body:          "",
			wantCode:      http.StatusBadRequest,
			authorization: true,
			tokenUserID:   "user-1",
		},
		{
			name:          "invalid url",
			method:        http.MethodPost,
			contentType:   "application/json",
			body:          `{"url":"http://"}`,
			wantCode:      http.StatusBadRequest,
			authorization: true,
			tokenUserID:   "user-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestHandler(t).URLRouter()
			req := httptest.NewRequest(tt.method, "/api/shorten", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			if tt.authorization {
				req.Header.Set("Authorization", "Bearer "+newAccessToken(t, tt.tokenUserID))
			}
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

func TestListUserURLsHandler(t *testing.T) {
	logger.InitTest()

	h := newTestHandler(t)
	if _, err := h.service.ShortenURL(context.Background(), "user-1", "https://google.com"); err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}
	if _, err := h.service.ShortenURL(context.Background(), "user-1", "https://openai.com"); err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}
	if _, err := h.service.ShortenURL(context.Background(), "user-2", "https://example.com"); err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}

	r := h.URLRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	req.Header.Set("Authorization", "Bearer "+newAccessToken(t, "user-1"))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var urls []dto.UserURLResponse
	if err := json.Unmarshal(w.Body.Bytes(), &urls); err != nil {
		t.Fatalf("response is not valid json: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %d", len(urls))
	}
	for _, url := range urls {
		if url.OriginalURL == "https://example.com" {
			t.Fatal("got URL from another user")
		}
	}
}

func TestDeleteURLHandler(t *testing.T) {
	logger.InitTest()

	h := newTestHandler(t)
	shortened, err := h.service.ShortenURL(context.Background(), "user-1", "https://google.com")
	if err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}

	r := h.URLRouter()

	forbiddenReq := httptest.NewRequest(http.MethodDelete, "/api/urls/"+shortened.UUID, nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+newAccessToken(t, "user-2"))
	forbiddenW := httptest.NewRecorder()
	r.ServeHTTP(forbiddenW, forbiddenReq)

	if forbiddenW.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", forbiddenW.Code, http.StatusForbidden)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/urls/"+shortened.UUID, nil)
	deleteReq.Header.Set("Authorization", "Bearer "+newAccessToken(t, "user-1"))
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", deleteW.Code, http.StatusNoContent)
	}

	redirectReq := httptest.NewRequest(http.MethodGet, "/"+shortened.UUID, nil)
	redirectW := httptest.NewRecorder()
	r.ServeHTTP(redirectW, redirectReq)

	if redirectW.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", redirectW.Code, http.StatusNotFound)
	}
}

func newTestHandler(t *testing.T) *ShortenHandler {
	t.Helper()

	repo := newStubURLRepository()
	svc := service.NewShortenerService(repo, "http://localhost:8080", nil)
	verifier, err := shortenerauth.NewVerifier(testAccessTokenSecret, testAccessTokenIssuer)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	return NewShortenHandler(svc, transportmiddleware.RequireAuth(verifier))
}

func newAccessToken(t *testing.T, userID string) string {
	t.Helper()

	claims := shortenerauth.AccessClaims{
		Email: userID + "@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    testAccessTokenIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testAccessTokenSecret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	return signed
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
