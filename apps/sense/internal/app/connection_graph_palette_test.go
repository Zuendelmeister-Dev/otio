package app

import (
	"os"
	"strings"
	"testing"
)

func TestSenseConnectionGraphUsesCommonPalette(t *testing.T) {
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"const BUILD_MARKER='v57-test-and-graph-fix-sense'",
		"const presenseColor=CONNECTION_PALETTE.presense",
		"const senseColor=CONNECTION_PALETTE.sense",
		"addLine(sourceAnchorX,y+36,senseX,senseY+38,value.connected,presenseColor,'presense')",
		"addLine(senseX+senseW,senseY+38,brokerX,brokerY+38,status.broker.connected,senseColor,'sense')",
		"addNode(brokerX,brokerY,brokerW,'system broker','MQTT Broker'",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %s", want)
		}
	}
}
