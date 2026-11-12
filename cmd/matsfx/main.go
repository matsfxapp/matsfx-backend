package main

import (
	"log/slog"
	"os"

	"github.com/matsfxapp/matsfx-backend/internal/config"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	slog.Info("matsfx backend is starting…", "addr", cfg.HTTPAddr, "origin", cfg.PublicOrigin)
}
