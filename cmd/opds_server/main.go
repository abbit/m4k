package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
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
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	server := &http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: server.New(ctx, providers),
	}

	go func() {
		log.Println("Listening on", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalln(err)
		}
	}()

	<-ctx.Done()
	log.Println("context done, shutting down a server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("while shutting down server: %w", err)
	}

	return nil
}
