// internal/server/sse_test.go
package server_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chandler767/redpanda-rec-room/internal/server"
)

func TestSSEBroker_PublishAndClose(t *testing.T) {
	broker := server.NewSSEBroker()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.ServeHTTP(rec, req)
	}()

	time.Sleep(20 * time.Millisecond)
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
