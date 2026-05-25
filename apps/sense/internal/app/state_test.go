package app

import (
	"testing"
	"time"
)

func TestHumanAge(t *testing.T) {
	_, human := humanAge("")
	if human != "n/a" {
		t.Fatalf("humanAge empty = %q", human)
	}
	stamp := time.Now().UTC().Add(-70 * time.Second).Format(time.RFC3339Nano)
	seconds, human := humanAge(stamp)
	if seconds < 60 || seconds > 90 {
		t.Fatalf("seconds = %d, want around 70", seconds)
	}
	if human == "n/a" {
		t.Fatalf("human age should be formatted")
	}
}

func TestAddLogLimitsEntries(t *testing.T) {
	state.Lock()
	state.Logs = nil
	state.Unlock()
	for i := 0; i < 510; i++ {
		addLog("INFO", "test", "entry")
	}
	state.Lock()
	length := len(state.Logs)
	state.Unlock()
	if length != 500 {
		t.Fatalf("log length = %d, want 500", length)
	}
}
