package main

import (
	"os"
	"strings"
	"testing"
)

func TestDispenseUIContainsRequiredPages(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{"Dashboard", "Quick Metrics", "Components", "Logs", "IoT Dispense"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected UI to contain %s", want)
		}
	}
}
