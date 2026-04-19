package eventcontroller

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

// EventFilter defines which events a handler wants to receive.
type EventFilter struct {
	Type    events.Type // container, service, node, network, etc.
	Actions []string    // start, stop, update, etc. Empty = all actions.
}

// Handler processes a Docker event.
type Handler func(ctx context.Context, event events.Message)

// registration pairs a filter with its handler.
type registration struct {
	filter  EventFilter
	handler Handler
}

// Controller listens to Docker events and dispatches to registered handlers.
type Controller struct {
	client        *client.Client
	registrations []registration
	mu            sync.RWMutex
	logger        *slog.Logger
}

// New creates a Controller with the given Docker client.
func New(cli *client.Client, logger *slog.Logger) *Controller {
	return &Controller{
		client: cli,
		logger: logger,
	}
}

// On registers a handler for events matching the filter.
func (c *Controller) On(filter EventFilter, handler Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.registrations = append(c.registrations, registration{filter: filter, handler: handler})
}

// Run starts listening to Docker events. Blocks until ctx is cancelled.
// Automatically reconnects on errors.
func (c *Controller) Run(ctx context.Context) error {
	for {
		if err := c.listen(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Error("event stream error, reconnecting in 5s", "error", err)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (c *Controller) listen(ctx context.Context) error {
	msgCh, errCh := c.client.Events(ctx, events.ListOptions{})
	c.logger.Info("connected to Docker event stream")

	for {
		select {
		case event := <-msgCh:
			c.dispatch(ctx, event)
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *Controller) dispatch(ctx context.Context, event events.Message) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, reg := range c.registrations {
		if !matches(reg.filter, event) {
			continue
		}
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					c.logger.Error("handler panic", "recover", r, "event_type", event.Type, "action", event.Action)
				}
			}()
			h(ctx, event)
		}(reg.handler)
	}
}

func matches(f EventFilter, e events.Message) bool {
	if f.Type != "" && f.Type != e.Type {
		return false
	}
	if len(f.Actions) == 0 {
		return true
	}
	for _, a := range f.Actions {
		if events.Action(a) == e.Action {
			return true
		}
	}
	return false
}
