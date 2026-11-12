package db

import (
	"context"
	"strings"
	"testing"
)

func TestOpenInvalidDSN(t *testing.T) {
	_, err := Open(context.Background(), "postgres://matsfx:secret@localhost:notaport/matsfx")
	if err == nil || !strings.HasPrefix(err.Error(), "parse database url:") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestOpenUnreachable(t *testing.T) {
	_, err := Open(context.Background(), "postgres://matsfx:secret@127.0.0.1:1/matsfx?connect_timeout=1")
	if err == nil || !strings.HasPrefix(err.Error(), "ping database:") {
		t.Fatalf("expected ping error, got %v", err)
	}
}

func TestOpenErrorHidesPassword(t *testing.T) {
	_, err := Open(context.Background(), "postgres://matsfx:password@localhost:notaport/matsfx")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "password") {
		t.Fatalf("error leaks the password: %v", err)
	}
}
