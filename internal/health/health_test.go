package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matsfxapp/matsfx-backend/internal/httpx"
)

func TestFauledCheckIsNamed(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	router := httpx.NewRouter(log)
	router.Handle("GET /api/health", Handler(log,
		Check{"database", func(context.Context) error { return errors.New("dial tcp 10.0.0.5:5432: connection refused") }},
		Check{"storage", func(context.Context) error { return nil }},
	))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Fatalf("check error leaked: %s", rec.Body)
	}
	var rep report
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if rep.Status != "unavailable" || rep.Checks["database"] != "unavailable" || rep.Checks["storage"] != "ok" {
		t.Fatalf("report %+v", rep)
	}
	if !strings.Contains(logs.String(), "10.0.0.5") || !strings.Contains(logs.String(), "check=database") {
		t.Fatalf("check error not logged: %s", logs.String())
	}
}
