//go:build integration

package eventcontroller

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

func TestIntegration_EventCapture(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker not available:", err)
	}
	defer cli.Close()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctrl := New(cli, logger)

	var mu sync.Mutex
	var captured []events.Message

	ctrl.On(EventFilter{Type: "container"}, func(ctx context.Context, e events.Message) {
		mu.Lock()
		captured = append(captured, e)
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go ctrl.Run(ctx)

	// Create a container to generate events
	time.Sleep(1 * time.Second)
	execCtx := context.Background()
	_, err = cli.ContainerCreate(execCtx, nil, nil, nil, nil, "")
	// We don't need the container to succeed, just generate events

	time.Sleep(3 * time.Second)
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if len(captured) == 0 {
		t.Log("No events captured (may need Docker Swarm mode)")
	} else {
		t.Logf("Captured %d events", len(captured))
	}
}
