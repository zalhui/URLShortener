package middleware

import (
	"net/http"
	"time"

	"github.com/zalhui/URLShortener/auth/internal/logger"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	data *responseData
}

func (r *loggingResponseWriter) WriteHeader(status int) {
	r.data.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.data.size += size
	return size, err
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &loggingResponseWriter{
			ResponseWriter: w,
			data:           &responseData{},
		}

		next.ServeHTTP(wrapped, r)

		status := wrapped.data.status
		if status == 0 {
			status = http.StatusOK
		}

		logger.Sugar.Infow(
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"size", wrapped.data.size,
			"duration", time.Since(start),
		)
	})
}
