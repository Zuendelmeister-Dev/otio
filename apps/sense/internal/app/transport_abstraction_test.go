package app

import (
	"os"
	"strings"
	"testing"
)

func TestSenseUsesTransportAbstraction(t *testing.T) {
	content, err := os.ReadFile("sense_transport.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, want := range []string{
		"type EventPublisher interface",
		"PublishJSON(topic string, payload any) error",
		"type mqttEventPublisher struct",
		"func connectTransport(config Config)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("transport abstraction missing %s", want)
		}
	}
}
