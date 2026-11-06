package httpapi

import (
	"net/http"

	"github.com/rktkov/ytfeed/internal/pkg/trace"
)

type tracedResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *tracedResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Flush keeps SSE working through the trace wrapper.
func (w *tracedResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// TraceMiddleware records one HTTP span per request.
func TraceMiddleware(logger *trace.Logger, next http.Handler) http.Handler {
	if logger == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := logger.Start(r.Context(), "http.request", map[string]any{
			"method": r.Method, "path": r.URL.Path,
		})
		traced := &tracedResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(traced, r)
		span.Attributes["status"] = traced.status
		span.End(nil)
	})
}
