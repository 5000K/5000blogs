package server

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder captures status code and bytes written.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	// Default status to 200 if WriteHeader wasn't called explicitly.
	if r.status == 0 {
		r.status = http.StatusOK
	}

	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// RequestLogger returns a chi-compatible middleware that logs each request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rr := &responseRecorder{
				ResponseWriter: w,
			}

			next.ServeHTTP(rr, r)

			// If nothing was written at all, assume 200 OK.
			if rr.status == 0 {
				rr.status = http.StatusOK
			}

			logger.Info("http request",
				slog.String("path", r.URL.Path),
				slog.Int("status", rr.status),
				slog.Int("bytes_sent", rr.bytes),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
