package eventcontroller

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/events"
)

func TestMatches(t *testing.T) {
	tests := []struct {
		name   string
		filter EventFilter
		event  events.Message
		want   bool
	}{
		{"empty filter matches all", EventFilter{}, events.Message{Type: "container", Action: "start"}, true},
		{"type match", EventFilter{Type: "container"}, events.Message{Type: "container", Action: "start"}, true},
		{"type mismatch", EventFilter{Type: "service"}, events.Message{Type: "container", Action: "start"}, false},
		{"action match", EventFilter{Type: "container", Actions: []string{"start", "stop"}}, events.Message{Type: "container", Action: "start"}, true},
		{"action mismatch", EventFilter{Type: "container", Actions: []string{"stop"}}, events.Message{Type: "container", Action: "start"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matches(tt.filter, tt.event); got != tt.want {
				t.Errorf("matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDispatch(t *testing.T) {
	logger := slog.Default()
	c := &Controller{logger: logger}

	var mu sync.Mutex
	var received []string

	c.On(EventFilter{Type: "container", Actions: []string{"start"}}, func(ctx context.Context, e events.Message) {
		mu.Lock()
		received = append(received, "container-start")
		mu.Unlock()
	})
	c.On(EventFilter{Type: "service"}, func(ctx context.Context, e events.Message) {
		mu.Lock()
		received = append(received, "service-any")
		mu.Unlock()
	})

	ctx := context.Background()

	// Should trigger container-start handler only
	c.dispatch(ctx, events.Message{Type: "container", Action: "start"})
	// Should trigger service-any handler
	c.dispatch(ctx, events.Message{Type: "service", Action: "update"})
	// Should trigger nothing
	c.dispatch(ctx, events.Message{Type: "node", Action: "down"})

	time.Sleep(50 * time.Millisecond) // handlers run in goroutines

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 dispatches, got %d: %v", len(received), received)
	}
}
