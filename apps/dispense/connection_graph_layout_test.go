package main

import (
	"os"
	"strings"
	"testing"
)

func TestDispenseConnectionGraphHasVisibleNodeLayout(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		".connection-graph{position:relative;min-height:320px;height:320px",
		".node{position:absolute",
		"build v60-dispense-test-alignment",
		"addNode(brokerX,brokerY,brokerW,'system','Input MQTT Broker'",
		"addNode(dispenseX,dispenseY,dispenseW,'dispense',instance",
		"addNode(targetX,targetY,targetW,'system','Target MQTT Broker'",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("dispense graph layout should contain %s", want)
		}
	}
}
