// internal/server/sse.go
package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// SSEBroker streams log messages to connected HTTP clients via Server-Sent Events.
type SSEBroker struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
	done    chan struct{}
}

// NewSSEBroker creates a ready-to-use SSEBroker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: map[chan string]struct{}{},
		done:    make(chan struct{}),
	}
}

// Publish sends msg to all connected clients. Drops messages to slow clients.
// Newlines are replaced with spaces to preserve SSE framing.
func (b *SSEBroker) Publish(msg string) {
	msg = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(msg)
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// SubscriberCount returns the number of currently connected SSE clients.
func (b *SSEBroker) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.clients)
}

// Close signals all clients to disconnect.
func (b *SSEBroker) Close() {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
}

// ServeHTTP implements http.Handler for SSE streaming.
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.clients, ch)
		b.mu.Unlock()
	}()

	flusher, canFlush := w.(http.Flusher)
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if canFlush {
				flusher.Flush()
			}
		case <-b.done:
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if canFlush {
				flusher.Flush()
			}
			return
		case <-r.Context().Done():
			return
		}
	}
}
