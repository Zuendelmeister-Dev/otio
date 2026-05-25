package main

import (
	"os"
	"strings"
	"testing"
)

func TestDispenseConnectionViewsUseSharedPalette(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"const CONNECTION_PALETTE={presense:'#8a9099',sense:'#14b8a6',lense:'#ff8a1d',dispense:'#ef4444',system:'#facc15',down:'#ff4d4f'}",
		"color:CONNECTION_PALETTE.system",
		"color:CONNECTION_PALETTE.dispense",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("dispense connection views should use shared palette entry: %s", want)
		}
	}
}
