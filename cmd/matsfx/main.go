package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/matsfxapp/matsfx-backend/internal/config"
	"github.com/matsfxapp/matsfx-backend/internal/db"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	slog.Info("matsfx backend is starting", "addr", cfg.HTTPAddr, "origin", cfg.PublicOrigin)

	database, err := db.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("database connected")
}
