package main

import (
	"os"
	"strings"
	"testing"
)

func TestPresenseSimulationAndExternalSourcesAreRenderedDifferently(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	mustContain := []string{
		"const LENSE_MENU_BUILD_MARKER='v65-presense-simulation-vs-external-source'",
		"function isPresenseSimulationSource(source)",
		"function sourceNodeClass(source)",
		"function sourceNodeMenuItem(source)",
		"kind: 'IoT Presense (Simulation)'",
		"const sourceCls=sourceNodeClass(a)+(sourceConnected?'':' down-node')",
		"node(sourceX,y,sourceW,sourceCls,a.agentId,(sourceConnected?'connected':'down')+' | '+a.sourceType,sourceNodeMenuItem(a))",
		".node.source-external{border-color:#8a9099;cursor:default}",
		".node.presense-simulation{border-color:#8a9099;border-style:dashed}",
		".node.down-node{border-color:var(--red)!important;border-style:dashed!important}",
		"el.className='node '+(cls||'')+(item?' clickable':'')",
	}

	for _, want := range mustContain {
		if !strings.Contains(content, want) {
			t.Fatalf("expected source rendering behavior not found: %s", want)
		}
	}

	if strings.Contains(content, "kind:'IoT Presense',url:a.url") {
		t.Fatalf("source endpoints must not always be treated as IoT Presense")
	}
}
