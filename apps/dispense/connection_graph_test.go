package main

import (
	"os"
	"strings"
	"testing"
)

func TestDispenseConnectionGraphExistsAndUsesCommonPalette(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"build v60-dispense-test-alignment",
		"id=\"dispenseConnectionGraph\"",
		"id=\"dispenseComponentsConnectionGraph\"",
		"function renderDispenseConnectionGraph(targetId)",
		"const systemColor=CONNECTION_PALETTE.system",
		"const dispenseColor=CONNECTION_PALETTE.dispense",
		"addLine(brokerX+brokerW,brokerY+38,dispenseX,dispenseY+38,Boolean(input.connected),systemColor,'system')",
		"addLine(dispenseX+dispenseW,dispenseY+38,targetX,targetY+38,Boolean(output.connected),dispenseColor,'dispense')",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %s", want)
		}
	}
}
