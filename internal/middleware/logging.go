package middleware

import (
	"net/http"
	"time"

	"github.com/zalhui/URLShortener/internal/logger"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		data *responseData
	}
)

func (r *loggingResponseWriter) WriteHeader(status int) {
	r.data.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.data.size = size
	return size, err
}

func LoggingMidlleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wr := &loggingResponseWriter{
			ResponseWriter: w,
			data: &responseData{
				status: 0,
				size:   0,
			},
		}

		next.ServeHTTP(wr, r)

		duration := time.Since(start)

		logger.Sugar.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", wr.data.status,
			"size", wr.data.size,
			"duration", duration,
		)

	})
}
