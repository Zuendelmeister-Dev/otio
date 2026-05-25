package main

import (
	"os"
	"strings"
	"testing"
)

func TestAgentViewUsesConfigurationLikeGroups(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	for _, want := range []string{
		".agent-group.presense{border-left-color:#8a9099}",
		".agent-group.sense{border-left-color:#14b8a6}",
		".agent-group.lense{border-left-color:var(--accent)}",
		".agent-group.dispense{border-left-color:#ef4444}",
		".agent-group.system{border-left-color:var(--system)}",
		"const groups=[",
		"title:'IoT Presense'",
		"title:'IoT Sense'",
		"title:'IoT Lense'",
		"title:'IoT Dispense'",
		"title:'System Components'",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("agent view should contain %s", want)
		}
	}

	for _, notWant := range []string{
		".agent-card.presense{border-left-color",
		".agent-card.sense{border-left-color",
		".agent-card.lense{border-left-color",
		".agent-card.dispense{border-left-color",
		".agent-card.system{border-left-color",
		"agent-card '+group.className",
	} {
		if strings.Contains(content, notWant) {
			t.Fatalf("agent cards must stay neutral and must not contain %s", notWant)
		}
	}
}

func TestSystemComponentsUseDedicatedYellow(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	for _, want := range []string{
		"--system:#facc15",
		".subnav a.system{border-left:3px solid var(--system)}",
		"const componentColor=CONNECTION_PALETTE.system",
		"const brokerCls='system'+(brokerOk?'':' down-node')",
		"const dbCls='system'+(dbOk?'':' down-node')",
		"node(brokerX,brokerY,brokerW,brokerCls,'MQTT Broker'",
		"node(dbX,dbY,dbW,dbCls,'Postgres'",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("system component color should contain %s", want)
		}
	}
}
