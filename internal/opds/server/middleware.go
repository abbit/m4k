package server

import (
	"log/slog"
	"net/http"
	"time"
)

// Custom ResponseWriter wrapper to capture the status code
type responseWriterWrapper struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	bodyBytes   int
}

func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	ww := &responseWriterWrapper{ResponseWriter: w}
	ww.status = http.StatusOK // default status code
	return ww
}

func (w *responseWriterWrapper) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
	w.wroteHeader = true
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bodyBytes += n
	return n, err
}

func logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wWrapper := newResponseWriterWrapper(w)

		next.ServeHTTP(wWrapper, r)

		duration := time.Since(start)
		status := wWrapper.status

		slog.Info("request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("status", status),
			slog.Int("body_bytes", wWrapper.bodyBytes),
			slog.Float64("duration_ms", duration.Seconds()*1000),
		)
	})
}
