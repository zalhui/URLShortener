package middleware

import (
	"net/http"
	"strings"

	authpkg "github.com/zalhui/URLShortener/internal/auth"
)

type AccessTokenVerifier interface {
	VerifyAccessToken(token string) (*authpkg.Identity, error)
}

func RequireAuth(verifier AccessTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if verifier == nil {
				http.Error(w, "authentication is unavailable", http.StatusInternalServerError)
				return
			}

			token, err := bearerTokenFromRequest(r)
			if err != nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			identity, err := verifier.VerifyAccessToken(token)
			if err != nil || identity == nil || identity.UserID == "" {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := WithUserID(r.Context(), identity.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
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
