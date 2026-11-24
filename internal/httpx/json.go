package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %s: %s", e.Status, e.Code, e.Message)
}

var errInternal = &Error{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Something went wrong."}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	return nil
}

func (rt *Router) writeError(w http.ResponseWriter, r *http.Request, err error) {
	id := RequestID(r.Context())
	if sw, ok := w.(*statusWriter); ok && sw.wroteHeader {
		rt.log.Error("handler failed after response started", "request_id", id, "err", err)
		return
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		rt.log.Error("handler failed", "request_id", id, "err", err)
		apiErr = errInternal
	}
	writeErrorBody(w, apiErr)
}

func writeErrorBody(w http.ResponseWriter, e *Error) {
	type body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	// status is already sent, a failed write means the client is gone
	_ = WriteJSON(w, e.Status, map[string]body{"error": {Code: e.Code, Message: e.Message}})
}
