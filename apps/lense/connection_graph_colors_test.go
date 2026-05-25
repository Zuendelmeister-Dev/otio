package main

import (
	"os"
	"strings"
	"testing"
)

func TestConnectionGraphPaletteIsConsistent(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"const CONNECTION_PALETTE={presense:'#8a9099',sense:'#14b8a6',lense:'#ff8a1d',dispense:'#ef4444',system:'#facc15',down:'#ff4d4f'}",
		"const presenseColor=CONNECTION_PALETTE.presense",
		"const senseColor=CONNECTION_PALETTE.sense",
		"const componentColor=CONNECTION_PALETTE.system",
		"const lenseColor=CONNECTION_PALETTE.lense",
		"const dispenseColor=CONNECTION_PALETTE.dispense",
		"line(sourceX+sourceW,sourceYs[i],senseX,senseY+42,Boolean(a.connected)&&group.connected,presenseColor)",
		"line(senseX+senseW,item.y,brokerX,brokerY+42,ok,senseColor)",
		"line(brokerX+brokerW,brokerY+42,lenseX,lenseY+42,brokerOk&&lenseOk,componentColor)",
		"line(brokerX+brokerW,brokerY+42,dispenseX,dispenseCenterY,summary.mqtt.connected&&dispenseOk,componentColor)",
		"line(dispenseX+dispenseW,dispenseCenterY,targetX,targetY+42,dispenseOk,dispenseColor)",
		"legendLine(CONNECTION_PALETTE.down,'Unavailable connection',true)",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing shared connection graph palette or usage: %s", want)
		}
	}
}
