package main

import (
	"os"
	"strings"
	"testing"
)

func TestConnectionGraphUsesForwardFlowLayout(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"const brokerX=1050",
		"const lenseX=1360",
		"const dispenseX=1460",
		"line(brokerX+brokerW,brokerY+42,lenseX,lenseY+42",
		"line(brokerX+brokerW,brokerY+42,dispenseX,dispenseCenterY",
		"line(sourceX+sourceW,sourceYs[i],senseX,senseY+42,Boolean(a.connected)&&group.connected,presenseColor",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("connection graph layout must contain %s", want)
		}
	}
}
