package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zalhui/URLShortener/auth/internal/config"
	"github.com/zalhui/URLShortener/auth/internal/delivery/http/dto"
	"github.com/zalhui/URLShortener/auth/internal/entity"
	"github.com/zalhui/URLShortener/auth/internal/service"
)

const maxRequestBodySize = 1 << 20

type AuthUseCase interface {
	Ping(ctx context.Context) error
	Register(ctx context.Context, email, password string, meta service.RequestMeta) (*service.AuthResult, error)
	Login(ctx context.Context, email, password string, meta service.RequestMeta) (*service.AuthResult, error)
	Refresh(ctx context.Context, rawRefreshToken string, meta service.RequestMeta) (*service.AuthResult, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	CurrentUser(ctx context.Context, accessToken string) (*entity.User, error)
}

type AuthHandler struct {
	service AuthUseCase
	cookie  config.CookieConfig
}

func NewAuthHandler(service AuthUseCase, cookie config.CookieConfig) *AuthHandler {
	return &AuthHandler{
		service: service,
		cookie:  cookie,
	}
}

func (h *AuthHandler) Router() chi.Router {
	r := chi.NewRouter()
	r.Post("/auth/register", h.registerHandler)
	r.Post("/auth/login", h.loginHandler)
	r.Post("/auth/refresh", h.refreshHandler)
	r.Post("/auth/logout", h.logoutHandler)
	r.Get("/auth/me", h.meHandler)
	return r
}

func (h *AuthHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		http.Error(w, "database is unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		return
	}

	result, err := h.service.Register(r.Context(), req.Email, req.Password, requestMeta(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	h.writeAuthResponse(w, http.StatusCreated, result)
}

func (h *AuthHandler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		return
	}

	result, err := h.service.Login(r.Context(), req.Email, req.Password, requestMeta(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	h.writeAuthResponse(w, http.StatusOK, result)
}

func (h *AuthHandler) refreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := h.readRefreshCookie(r)
	if err != nil {
		http.Error(w, "refresh token is missing", http.StatusUnauthorized)
		return
	}

	result, err := h.service.Refresh(r.Context(), refreshToken, requestMeta(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	h.writeAuthResponse(w, http.StatusOK, result)
}

func (h *AuthHandler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, _ := h.readRefreshCookie(r)
	if err := h.service.Logout(r.Context(), refreshToken); err != nil {
		h.writeServiceError(w, err)
		return
	}

	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) meHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := bearerTokenFromRequest(r)
	if err != nil {
		http.Error(w, "access token is missing", http.StatusUnauthorized)
		return
	}

	user, err := h.service.CurrentUser(r.Context(), accessToken)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

func (h *AuthHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidEmail), errors.Is(err, service.ErrWeakPassword):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrEmailTaken):
		http.Error(w, "email already taken", http.StatusConflict)
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrUnauthorized):
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) writeAuthResponse(w http.ResponseWriter, status int, result *service.AuthResult) {
	writeJSON(w, status, dto.AuthResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(result.AccessTokenExpiresAt).Seconds()),
		User: dto.UserResponse{
			ID:        result.User.ID,
			Email:     result.User.Email,
			CreatedAt: result.User.CreatedAt,
		},
	})
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    refreshToken,
		Path:     "/auth",
		Domain:   h.cookie.Domain,
		HttpOnly: h.cookie.HTTPOnly,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(h.cookie.MaxAge.Seconds()),
		Expires:  time.Now().UTC().Add(h.cookie.MaxAge),
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    "",
		Path:     "/auth",
		Domain:   h.cookie.Domain,
		HttpOnly: h.cookie.HTTPOnly,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (h *AuthHandler) readRefreshCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(h.cookie.Name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func requestMeta(r *http.Request) service.RequestMeta {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	return service.RequestMeta{
		UserAgent: r.UserAgent(),
		IPAddress: host,
	}
}

func bearerTokenFromRequest(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", http.ErrNoCookie
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", http.ErrNoCookie
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", http.ErrNoCookie
	}

	return token, nil
}
