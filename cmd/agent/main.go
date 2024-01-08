package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/nginx/agent/v3/internal/apis/http"
)

func main() {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	dataplaneServer := http.NewDataplaneServer("0.0.0.0:8091", logger)
	dataplaneServer.Run(context.Background())
}
