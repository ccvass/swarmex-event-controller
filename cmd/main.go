package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	eventcontroller "github.com/ccvass/swarmex/swarmex-event-controller"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error("failed to create Docker client", "error", err)
		os.Exit(1)
	}
	defer cli.Close()

	ctrl := eventcontroller.New(cli, logger)

	// Log all events (default behavior when running standalone)
	ctrl.On(eventcontroller.EventFilter{}, func(ctx context.Context, e events.Message) {
		logger.Info("event",
			"type", e.Type,
			"action", e.Action,
			"actor_id", e.Actor.ID,
			"attributes", e.Actor.Attributes,
		)
	})

	// Health endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "ok")
		})
		addr := ":8080"
		logger.Info("health endpoint", "addr", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error("health server error", "error", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Info("swarmex-event-controller starting")
	if err := ctrl.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("controller stopped", "error", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
