package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abbit/m4k/internal/opds/server"
)

const port = "6333"

var providers = []string{
	"mango-mangapill",
	"mango-manganato",
	"mango-mangakakalot",
	"mango-mangadex",
	"mango-mangaplus",
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		slog.Error("while running", slog.Any("error", err))
	}
}

func run(ctx context.Context) error {
	server := &http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: server.New(ctx, providers),
	}

	go func() {
		slog.Info("Start listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("while listening", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("context done, shutting down a server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("while shutting down server: %w", err)
	}

	return nil
}
