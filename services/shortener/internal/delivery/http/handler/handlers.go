package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zalhui/URLShortener/internal/delivery/http/dto"
	transportmiddleware "github.com/zalhui/URLShortener/internal/delivery/http/middleware"
	"github.com/zalhui/URLShortener/internal/entity"
	"github.com/zalhui/URLShortener/internal/service"
)

const maxRequestBodySize = 1 << 20

type ShortenerUseCase interface {
	Ping(ctx context.Context) error
	ShortenURL(ctx context.Context, userID, originalURL string) (*entity.URL, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
}

type ShortenHandler struct {
	service ShortenerUseCase
}

func NewShortenHandler(service ShortenerUseCase) *ShortenHandler {
	return &ShortenHandler{
		service: service,
	}
}

func (h *ShortenHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		http.Error(w, "database is unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ShortenHandler) shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if !hasContentType(r, "text/plain") {
		http.Error(w, "unsupported Content-Type", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	originalURL, err := io.ReadAll(r.Body)
	if err != nil || len(originalURL) == 0 {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	userID := transportmiddleware.UserIDFromContext(r.Context())
	url, err := h.service.ShortenURL(r.Context(), userID, string(originalURL))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(url.ShortURL)); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func (h *ShortenHandler) getOriginalURLHandler(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "shortID")

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(w, "short url not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to resolve short url", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *ShortenHandler) jsonShortenHandler(w http.ResponseWriter, r *http.Request) {
	if !hasContentType(r, "application/json") {
		http.Error(w, "unsupported Content-Type", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req dto.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	userID := transportmiddleware.UserIDFromContext(r.Context())
	url, err := h.service.ShortenURL(r.Context(), userID, req.URL)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(dto.ShortenResponse{Result: url.ShortURL}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (h *ShortenHandler) URLRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/{shortID}", h.getOriginalURLHandler)
	r.Post("/", h.shortenURLHandler)
	r.Post("/api/shorten", h.jsonShortenHandler)

	return r
}

func (h *ShortenHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidURL):
		http.Error(w, "invalid url", http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func hasContentType(r *http.Request, expected string) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == expected
}
