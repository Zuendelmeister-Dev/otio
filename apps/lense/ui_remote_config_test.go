package main

import (
	"os"
	"strings"
	"testing"
)

func TestLenseConfigDetailUsesRemoteSenseHistory(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{"/api/remote-config/history", "supportsRemoteConfig", "Applied remotely", "Remote draft is valid"} {
		if !strings.Contains(content, want) {
			t.Fatalf("configuration detail must contain %s", want)
		}
	}
}

func TestConnectionGraphUsesWideDraggableCanvas(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{"IoTWorkspace.graphViewport", "overflow-x:auto", "boxWidth=1780", "/static/workspace-ui.js"} {
		if !strings.Contains(content, want) {
			t.Fatalf("connection graph must contain %s", want)
		}
	}
}
