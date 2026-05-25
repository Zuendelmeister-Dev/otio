package main

import (
	"strings"
	"testing"
	"time"
)

func TestNodeValueTemperature(t *testing.T) {
	value := nodeValue("ns=2;s=Machine.Temperature", time.Now(), 30.0, 1400.0, 8.0, "sine")
	if value.NodeID == "" {
		t.Fatal("expected node id")
	}
	if value.Quality != "good" {
		t.Fatalf("quality = %s, want good", value.Quality)
	}
	if value.Unit != "°C" {
		t.Fatalf("unit = %s, want °C", value.Unit)
	}
}

func TestNodeValueSpeed(t *testing.T) {
	value := nodeValue("ns=2;s=Machine.Speed", time.Now(), 30.0, 1400.0, 8.0, "sine")
	if !strings.Contains(value.Unit, "rpm") {
		t.Fatalf("unit = %s, want rpm", value.Unit)
	}
}

func TestNodeValueStateReturnsString(t *testing.T) {
	value := nodeValue("ns=2;s=Machine.State", time.Now(), 30.0, 1400.0, 8.0, "sine")
	if value.NodeID == "" {
		t.Fatal("expected node id")
	}
	if value.Quality != "good" {
		t.Fatalf("quality = %s, want good", value.Quality)
	}
	if _, ok := value.Value.(string); !ok {
		t.Fatalf("value type = %T, want string", value.Value)
	}
}

func TestSubscriptionPayloadContainsRequestedNodes(t *testing.T) {
	payload := subscriptionPayload([]string{"ns=2;s=Machine.Temperature", "ns=2;s=Machine.State"}, time.Now(), 30.0, 1400.0, 8.0, "sine")
	items, ok := payload["items"].([]NodeValue)
	if !ok {
		t.Fatalf("items type = %T, want []NodeValue", payload["items"])
	}
	if len(items) != 2 {
		t.Fatalf("items length = %d, want 2", len(items))
	}
	if items[0].NodeID != "ns=2;s=Machine.Temperature" {
		t.Fatalf("first node id = %s", items[0].NodeID)
	}
}

func TestSubscriptionIntervalHasLowerBound(t *testing.T) {
	if got := subscriptionInterval("100"); got != 250*time.Millisecond {
		t.Fatalf("interval = %s, want 250ms", got)
	}
	if got := subscriptionInterval("1500"); got != 1500*time.Millisecond {
		t.Fatalf("interval = %s, want 1500ms", got)
	}
}
