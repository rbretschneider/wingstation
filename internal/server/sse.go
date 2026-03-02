package server

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// SSEBroker manages Server-Sent Events connections.
type SSEBroker struct {
	mu          sync.RWMutex
	clients     map[chan string]struct{}
	stopCh      chan struct{}
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[chan string]struct{}),
		stopCh:  make(chan struct{}),
	}
}

// Start begins the broker (no-op for now, can add periodic tasks).
func (b *SSEBroker) Start() {}

// Stop closes all client connections.
func (b *SSEBroker) Stop() {
	close(b.stopCh)
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		close(ch)
	}
	b.clients = make(map[chan string]struct{})
}

// Subscribe adds a new client and returns its event channel.
func (b *SSEBroker) Subscribe() chan string {
	ch := make(chan string, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a client.
func (b *SSEBroker) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

// Broadcast sends a message to all connected clients.
// Multi-line data is split so each line gets its own "data:" prefix per SSE spec.
func (b *SSEBroker) Broadcast(eventName, data string) {
	var msg strings.Builder
	msg.WriteString("event: ")
	msg.WriteString(eventName)
	msg.WriteString("\n")
	for _, line := range strings.Split(data, "\n") {
		msg.WriteString("data: ")
		msg.WriteString(line)
		msg.WriteString("\n")
	}
	msg.WriteString("\n")
	formatted := msg.String()
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- formatted:
		default:
			// Client buffer full, skip
		}
	}
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send retry interval
	fmt.Fprintf(w, "retry: %d\n\n", s.cfg.SSERetryMs)
	flusher.Flush()

	ch := s.sse.Subscribe()
	defer s.sse.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprint(w, msg)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// watchDockerEvents listens to Docker events and broadcasts updates via SSE.
func (s *Server) watchDockerEvents() {
	ctx := context.Background()

	for {
		select {
		case <-s.sse.stopCh:
			return
		default:
		}

		eventFilter := filters.NewArgs()
		eventFilter.Add("type", "container")

		msgCh, errCh := s.dockerClient.Events(ctx, events.ListOptions{
			Filters: eventFilter,
		})

		// Also set up a periodic refresh ticker for stats
		ticker := time.NewTicker(time.Duration(s.cfg.StatsIntervalMs) * time.Millisecond)

		func() {
			defer ticker.Stop()
			for {
				select {
				case <-s.sse.stopCh:
					return
				case msg := <-msgCh:
					s.handleDockerEvent(msg)
				case err := <-errCh:
					if err != nil {
						slog.Error("Docker events stream error", "error", err)
					}
					return
				case <-ticker.C:
					s.broadcastContainerList()
				}
			}
		}()

		// Reconnect after a brief pause
		slog.Info("Reconnecting to Docker events stream...")
		time.Sleep(2 * time.Second)
	}
}

func (s *Server) handleDockerEvent(msg events.Message) {
	slog.Debug("Docker event", "action", msg.Action, "id", msg.Actor.ID)

	// Invalidate cache on any container event
	s.containerService.InvalidateCache()

	// Broadcast the updated container list
	s.broadcastContainerList()

	// Also send a targeted update for the specific container
	if msg.Actor.ID != "" {
		s.sse.Broadcast("container-update", fmt.Sprintf(`{"id":"%s","action":"%s"}`, msg.Actor.ID, msg.Action))
	}
}

func (s *Server) broadcastContainerList() {
	groups, err := s.containerService.ListGrouped(context.Background())
	if err != nil {
		slog.Error("SSE: listing containers for broadcast", "error", err)
		return
	}

	data := pageData{Groups: groups}
	var buf bytes.Buffer
	if err := s.partials.ExecuteTemplate(&buf, "container_list", data); err != nil {
		slog.Error("SSE: rendering container list", "error", err)
		return
	}

	s.sse.Broadcast("containers", buf.String())
}
