// Package httpx routes requests and writes JSON responses and errors
package httpx

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

type Router struct {
	mux *http.ServeMux
	log *slog.Logger
}

func NewRouter(log *slog.Logger) *Router {
	return &Router{mux: http.NewServeMux(), log: log}
}

func (rt *Router) Handle(pattern string, h HandlerFunc) {
	rt.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			rt.writeError(w, r, err)
		}
	})
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := rand.Text()
	w.Header().Set("X-Request-Id", id)
	r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id))
	sw := &statusWriter{ResponseWriter: w}

	defer func() {
		if p := recover(); p != nil {
			if p == http.ErrAbortHandler {
				panic(p)
			}
			rt.log.Error("panic", "request_id", id, "panic", p, "stack", string(debug.Stack()))
			if !sw.wroteHeader {
				writeErrorBody(sw, errInternal)
			}
		}
		rt.log.Info(
			"request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", sw.status, "duration", time.Since(start))
	}()

	// ServerMux would answer 404 and 405 as text/plain
	if h, pattern := rt.mux.Handler(r); pattern == "" {
		writeMuxError(sw, r, h)
		return
	}
	rt.mux.ServeHTTP(sw, r)
}

func writeMuxError(w http.ResponseWriter, r *http.Request, h http.Handler) {
	probe := &statusWriter{ResponseWriter: discardWriter{header: http.Header{}}}
	h.ServeHTTP(probe, r)

	if probe.status == http.StatusMethodNotAllowed {
		w.Header().Set("Allow", probe.Header().Get("Allow"))
		writeErrorBody(w, &Error{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: "Message not allowed."})
		return
	}
	writeErrorBody(w, &Error{Status: http.StatusNotFound, Code: "not_found", Message: "Not found."})
}

type requestIDKey struct{}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status = status
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type discardWriter struct {
	header http.Header
}

func (w discardWriter) Header() http.Header {
	return w.header
}

func (w discardWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
func (w discardWriter) WriteHeader(int) {}
