package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	defaultTimeout = 30 * time.Second
)

type contextKey string

const (
	RequestIDKey contextKey = "x-request-id"
	UserIDKey    contextKey = "user-id"
)

func ContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), defaultTimeout)
		defer cancel()

		requestID := generateRequestID()
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func generateRequestID() string {
	id := uuid.New().String()
	return id
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}
