package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atad-project/internal/jobs"
	"atad-project/internal/stats"
	"atad-project/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			return version.WriteJSON(os.Stdout)
		}
	}

	aliases, err := stats.LoadAliases(os.Getenv("GIT_STATS_ALIASES"))
	if err != nil {
		return fmt.Errorf("load identities: %w", err)
	}
	service := service{root: os.Getenv("GIT_STATS_ROOT"), jobs: jobs.New(aliases)}
	defer service.jobs.Reset()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	server := &http.Server{
		Addr:         ":8080",
		Handler:      service.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 6 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	failed := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", server.Addr, "commit", version.Current().Commit)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
