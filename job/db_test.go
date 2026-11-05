package job

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestClientOptionsFailBeforeHTTPWriteTimeout(t *testing.T) {
	// The HTTP server's WriteTimeout is 30s: selection must give up earlier for the 500 to reach the client.
	got := newClientOptions("mongodb://127.0.0.1:27017/").ServerSelectionTimeout
	if got == nil || *got >= 30*time.Second {
		t.Errorf("Expected a server selection timeout below 30s, got %v", got)
	}
}

func TestDBErrorMapsDuplicateKeyTo409(t *testing.T) {
	d := DB{ColName: "auth"}
	dup := mongo.WriteException{WriteErrors: []mongo.WriteError{{
		Code:    11000,
		Message: `E11000 duplicate key error collection: store.auth index: name_1 dup key: { name: "alice" }`,
	}}}

	res := d.dbError(dup)
	if res.Code != http.StatusConflict {
		t.Errorf("Expected 409, got %d", res.Code)
	}
	if strings.Contains(res.Message, "E11000") || strings.Contains(res.Message, "alice") {
		t.Errorf("Response leaks database details: %q", res.Message)
	}
}

func TestDBErrorHidesOtherErrors(t *testing.T) {
	d := DB{ColName: "product"}

	res := d.dbError(errors.New("connection reset by peer 10.0.0.5:27017"))
	if res.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", res.Code)
	}
	if strings.Contains(res.Message, "10.0.0.5") || strings.Contains(res.Message, "connection reset") {
		t.Errorf("Response leaks database details: %q", res.Message)
	}
}

func TestCommandMonitorDoesNotLogPayloads(t *testing.T) {
	m := newCommandMonitor()
	if m.Started != nil || m.Succeeded != nil {
		t.Error("command monitor must not log command or reply payloads")
	}
}

func TestCommandMonitorFailedDoesNotExit(t *testing.T) {
	m := newCommandMonitor()
	if m.Failed == nil {
		t.Fatal("command monitor must log failed commands")
	}

	// A failed command (network blip, bad skip value...) must not terminate the process.
	m.Failed(context.Background(), &event.CommandFailedEvent{
		CommandFinishedEvent: event.CommandFinishedEvent{CommandName: "find"},
		Failure:              errors.New("skip value must be non-negative"),
	})
}
