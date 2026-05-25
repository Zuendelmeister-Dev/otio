package app

import (
	"os"
	"strings"
	"testing"
)

func TestSenseConnectionGraphUsesSharedPalette(t *testing.T) {
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"const CONNECTION_PALETTE={presense:'#8a9099',sense:'#14b8a6',lense:'#ff8a1d',dispense:'#ef4444',system:'#facc15',down:'#ff4d4f'}",
		"const presenseColor=CONNECTION_PALETTE.presense",
		"const senseColor=CONNECTION_PALETTE.sense",
		"const systemColor=CONNECTION_PALETTE.system",
		"addLine(sourceAnchorX,y+36,senseX,senseY+38,value.connected,presenseColor)",
		"addLine(senseX+senseW,senseY+38,brokerX,brokerY+38,status.broker.connected,senseColor)",
		"addNode(sourceX,y,sourceW,'presense'",
		"addNode(senseX,senseY,senseW,'sense'",
		"addNode(brokerX,brokerY,brokerW,'system broker'",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing shared Sense connection graph palette or usage: %s", want)
		}
	}
}
