package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matsfxapp/matsfx-backend/internal/config"
	"github.com/matsfxapp/matsfx-backend/internal/db"
	"github.com/matsfxapp/matsfx-backend/internal/health"
	"github.com/matsfxapp/matsfx-backend/internal/httpx"
	"github.com/matsfxapp/matsfx-backend/internal/storage"
)

const shutdownTimeout = 8 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("matsfx backend stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	slog.Info("matsfx backend is starting", "addr", cfg.HTTPAddr, "origin", cfg.PublicOrigin)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer database.Close()
	slog.Info("database connected")

	store, err := storage.Open(ctx, cfg.S3)
	if err != nil {
		return fmt.Errorf("connect storage: %w", err)
	}
	slog.Info("storage connected", "bucket", cfg.S3.Bucket)

	router := httpx.NewRouter(slog.Default())
	router.Handle("GET /api/health", health.Handler(slog.Default(),
		health.Check{Name: "database", Run: database.PingContext},
		health.Check{Name: "storage", Run: store.CheckBucket},
	))

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.ListenAndServe()
	}()
	slog.Info("listening", "addr", cfg.HTTPAddr)

	select {
	case err := <-serveErr:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
	}
	stop()

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
