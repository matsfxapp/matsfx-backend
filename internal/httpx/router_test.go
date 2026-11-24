package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTestRouter(t *testing.T) (*Router, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	return NewRouter(slog.New(slog.NewTextHandler(&logs, nil))), &logs
}

func serve(rt *Router, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorBody {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type %q", ct)
	}
	var body errorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestHandleWritesJSON(t *testing.T) {
	rt, logs := newTestRouter(t)
	rt.Handle("GET /api/songs/{id}", func(w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id")})
	})

	rec := serve(rt, http.MethodGet, "/api/songs/abc")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"id":"abc"}` {
		t.Fatalf("body %s", got)
	}
	id := rec.Header().Get("X-Request-Id")
	if id == "" {
		t.Fatal("missing X-Request-Id")
	}
	if !strings.Contains(logs.String(), "request_id="+id) || !strings.Contains(logs.String(), "status=200") {
		t.Fatalf("access log missing request: %s", logs)
	}
}

func TestHandleAPIError(t *testing.T) {
	rt, _ := newTestRouter(t)
	rt.Handle("GET /api/songs/{id}", func(w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("load song: %w", &Error{Status: http.StatusNotFound, Code: "song_not_found", Message: "Song not found."})
	})

	rec := serve(rt, http.MethodGet, "/api/songs/abc")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	body := decodeError(t, rec)
	if body.Error.Code != "song_not_found" || body.Error.Message != "Song not found." {
		t.Fatalf("body %+v", body)
	}
}

func TestHandleInternalErrorIsHidden(t *testing.T) {
	rt, logs := newTestRouter(t)
	rt.Handle("GET /api/fail", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("connection to 10.0.0.5 refused")
	})

	rec := serve(rt, http.MethodGet, "/api/fail")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Fatalf("internal error leaked: %s", rec.Body)
	}
	if body := decodeError(t, rec); body.Error.Code != "internal_error" {
		t.Fatalf("body %+v", body)
	}
	if !strings.Contains(logs.String(), "10.0.0.5") {
		t.Fatalf("internal error not logged: %s", logs)
	}
}

func TestPanicIsRecovered(t *testing.T) {
	rt, logs := newTestRouter(t)
	rt.Handle("GET /api/panic", func(w http.ResponseWriter, r *http.Request) error {
		panic("boom")
	})

	rec := serve(rt, http.MethodGet, "/api/panic")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if body := decodeError(t, rec); body.Error.Code != "internal_error" {
		t.Fatalf("body %+v", body)
	}
	if !strings.Contains(logs.String(), "boom") || !strings.Contains(logs.String(), "status=500") {
		t.Fatalf("panic not logged: %s", logs)
	}
}

func TestErrorAfterResponseStarted(t *testing.T) {
	rt, logs := newTestRouter(t)
	rt.Handle("GET /api/partial", func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return errors.New("stream broke")
	})

	rec := serve(rt, http.MethodGet, "/api/partial")

	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("response changed after start: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(logs.String(), "stream broke") {
		t.Fatalf("error not logged: %s", logs)
	}
}

func TestUnknownPath(t *testing.T) {
	rt, _ := newTestRouter(t)
	rt.Handle("GET /api/songs", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	rec := serve(rt, http.MethodGet, "/api/imadethatbfamous")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	if body := decodeError(t, rec); body.Error.Code != "not_found" {
		t.Fatalf("body %+v", body)
	}
}

func TestWrongMethod(t *testing.T) {
	rt, _ := newTestRouter(t)
	rt.Handle("GET /api/songs", func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	rec := serve(rt, http.MethodDelete, "/api/songs")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Fatalf("Allow header %q", allow)
	}
	if body := decodeError(t, rec); body.Error.Code != "method_not_allowed" {
		t.Fatalf("body %+v", body)
	}
}
