package main

import (
	"os"
	"strings"
	"testing"
)

func TestConnectionGraphDownNodesAndLegend(t *testing.T) {
	content, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatalf("read static index: %v", err)
	}
	html := string(content)

	required := []string{
		`id="connectionGraphLegend"`,
		`function renderConnectionGraphLegend()`,
		`legendNode(CONNECTION_PALETTE.down,'Unavailable node',true)`,
		`legendLine(CONNECTION_PALETTE.down,'Unavailable connection',true)`,
		`.node.down-node`,
		`sourceCls=sourceNodeClass(a)+(sourceConnected?'':' down-node')`,
		`const senseCls='sense'+(group.connected?'':' down-node')`,
		`const brokerCls='system'+(brokerOk?'':' down-node')`,
		`const dispenseCls='dispense'+(dispenseOk?'':' down-node')`,
		`renderConnectionGraphLegend();`,
	}

	for _, item := range required {
		if !strings.Contains(html, item) {
			t.Fatalf("connection graph should contain %s", item)
		}
	}
}
