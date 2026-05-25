package main

import (
	"os"
	"strings"
	"testing"
)

func TestLenseConnectionGraphUsesComponentPalette(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"const BUILD_MARKER='v57-test-and-graph-fix-lense'",
		"const CONNECTION_PALETTE={presense:'#8a9099',sense:'#14b8a6',lense:'#ff8a1d',dispense:'#ef4444',system:'#facc15',down:'#ff4d4f'}",
		"const presenseColor=CONNECTION_PALETTE.presense",
		"const senseColor=CONNECTION_PALETTE.sense",
		"line(sourceX+sourceW,sourceYs[i],senseX,senseY+42,Boolean(a.connected)&&group.connected,presenseColor)",
		"line(senseX+senseW,item.y,brokerX,brokerY+42,ok,senseColor)",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("lense connection graph should contain %s", want)
		}
	}
}
