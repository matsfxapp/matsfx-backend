// Package health reports the reachability
package health

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/matsfxapp/matsfx-backend/internal/httpx"
)

const checkTimeout = 2 * time.Second

type Check struct {
	Name string
	Run  func(context.Context) error
}

type report struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func Handler(log *slog.Logger, checks ...Check) httpx.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)
		defer cancel()

		errs := make([]error, len(checks))
		var wg sync.WaitGroup
		for i, check := range checks {
			wg.Go(func() {
				errs[i] = check.Run(ctx)
			})
		}
		wg.Wait()

		rep := report{Status: "ok", Checks: make(map[string]string, len(checks))}
		status := http.StatusOK
		for i, check := range checks {
			if errs[i] != nil {
				log.Error("health check failed", "request_id", httpx.RequestID(r.Context()), "check", check.Name, "err", errs[i])
				rep.Checks[check.Name] = "unavailable"
				rep.Status = "unavailable"
				status = http.StatusServiceUnavailable
				continue
			}
			rep.Checks[check.Name] = "ok"
		}
		return httpx.WriteJSON(w, status, rep)
	}
}
