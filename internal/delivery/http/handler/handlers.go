package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/zalhui/URLShortener/internal/delivery/http/middleware"
	"github.com/zalhui/URLShortener/internal/entity"
	"github.com/zalhui/URLShortener/internal/service"
)

type ShortenHandler struct {
	service *service.ShortenerService
}

func NewShortenHandler(service *service.ShortenerService) *ShortenHandler {
	return &ShortenHandler{
		service: service,
	}
}

func (h *ShortenHandler) shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Unsupported Content-Type", http.StatusBadRequest)
		return
	}

	originalURL, err := io.ReadAll(r.Body)
	if err != nil || len(originalURL) == 0 {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	url, err := h.service.ShortenURL(string(originalURL))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(url.ShortURL))
}

func (h *ShortenHandler) getOriginalURLHandler(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "shortID")

	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	//ensure that the original URL starts with http:// or https://
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		originalURL = "https://" + originalURL
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *ShortenHandler) jsonShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("content-type") != "application/json" {
		http.Error(w, "Unsupported content-type", http.StatusBadRequest)
		return
	}

	var req entity.ShortenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := h.service.ShortenURL(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entity.ShortenResponse{Result: url.ShortURL})

}

func (h *ShortenHandler) URLRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.LoggingMidlleware)
	r.Get("/{shortID}", h.getOriginalURLHandler)
	r.Post("/", h.shortenURLHandler)
	r.Post("/api/shorten", h.jsonShortenHandler)
	return r
}
