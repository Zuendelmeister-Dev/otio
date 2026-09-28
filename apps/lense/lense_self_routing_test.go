package main

import (
	"os"
	"strings"
	"testing"
)

func TestLenseSelfNavigationDoesNotCreateRedundantLensePages(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	mustContain := []string{
		"const LENSE_MENU_BUILD_MARKER='v65-presense-simulation-vs-external-source'",
		`<a href="#dashboard" class="lense" data-lense-self-target="agents">IoT Lense</a>`,
		`<a href="#configurations/iot-lense" class="lense" data-lense-self-target="configurations">IoT Lense</a>`,
		"function isLenseSelfItem(item)",
		"IoTWorkspace.setTopology(topology)",
		"const remoteButton=(item.url&&!isLenseSelfItem(item))?",
		"if(page==='agents'&&id==='lense')",
		"link:'#dashboard',hint:'Analytics, historian and central view'",
	}

	for _, want := range mustContain {
		if !strings.Contains(content, want) {
			t.Fatalf("expected lense self routing marker or behavior not found: %s", want)
		}
	}

	if strings.Contains(content, `<a href="#agents/lense" class="lense">IoT Lense</a>`) {
		t.Fatalf("agents submenu must not route to a redundant IoT Lense-only agent page")
	}
}
