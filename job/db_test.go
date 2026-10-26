package job

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/event"
)

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
