package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShortenURLHandler(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantCode    int
	}{
		{
			name:        "successful shorten",
			method:      "POST",
			contentType: "text/plain",
			body:        "https://google.com",
			wantCode:    http.StatusCreated,
		},
		{
			name:        "wrong method",
			method:      "GET",
			contentType: "text/plain",
			body:        "https://google.com",
			wantCode:    http.StatusMethodNotAllowed,
		},
		{
			name:        "wrong content type",
			method:      "POST",
			contentType: "application/json",
			body:        "https://google.com",
			wantCode:    http.StatusBadRequest,
		},
		{
			name:        "empty body",
			method:      "POST",
			contentType: "text/plain",
			body:        "",
			wantCode:    http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			us := NewURLShortener()

			r := us.URLRouter()

			req, err := http.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.wantCode {
				t.Errorf("shortenURLHandler() = %v, want %v", w.Code, tt.wantCode)
			}
		})
	}
}

func TestGetOriginalURLHandler(t *testing.T) {
	shortener := NewURLShortener()

	shortener.urls["123"] = "https://google.com"

	tests := []struct {
		name     string
		method   string
		id       string
		wantCode int
	}{
		{
			name:     "successful get",
			method:   "GET",
			id:       "123",
			wantCode: http.StatusTemporaryRedirect,
		},
		{
			name:     "wrong method",
			method:   "POST",
			id:       "123",
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			name:     "wrong id",
			method:   "GET",
			id:       "456",
			wantCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := shortener.URLRouter()

			req, err := http.NewRequest(tt.method, fmt.Sprintf("/%s", tt.id), nil)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.wantCode {
				t.Errorf("getOriginalURLHandler() = %v, want %v", w.Code, tt.wantCode)
			}
		})
	}
}
