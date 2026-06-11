// internal/server/sse_test.go
package server_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chandler767/redpanda-rec-room/internal/server"
)

// waitSubscribers blocks until the broker has at least n subscribers or the deadline passes.
func waitSubscribers(t *testing.T, b *server.SSEBroker, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if b.SubscriberCount() >= n {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d subscriber(s)", n)
}

func TestSSEBroker_PublishAndClose(t *testing.T) {
	broker := server.NewSSEBroker()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.ServeHTTP(rec, req)
	}()

	waitSubscribers(t, broker, 1)
	broker.Publish("hello world")
	broker.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeHTTP did not return after Close()")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "hello world") {
		t.Errorf("body missing 'hello world': %q", body)
	}
}
