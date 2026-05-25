package main

import (
	"os"
	"strings"
	"testing"
)

func TestLenseDashboardDoesNotReferenceMissingMinMaxElements(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if strings.Contains(content, "min.textContent") || strings.Contains(content, "max.textContent") {
		t.Fatalf("loadSummary must not write to missing min/max elements because this stops dashboard rendering")
	}
}

func TestLenseUsesSharedSelectRenderer(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "IoTStandardChart.fillSelect") {
		t.Fatalf("Lense must reuse the shared select renderer")
	}
}

func TestLensePagesArrayReferencesExistingSections(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, page := range []string{"dashboard", "agents", "agent-detail", "analytics", "namespace", "components", "logs"} {
		if !strings.Contains(content, `id="page-`+page+`"`) {
			t.Fatalf("missing page section for %s", page)
		}
	}
	for _, removed := range []string{"broker", "database"} {
		if strings.Contains(content, `page-`+removed) {
			t.Fatalf("%s must be represented through Components view, not as a separate page section", removed)
		}
	}
	if strings.Contains(content, "'broker','database'") || strings.Contains(content, "\"broker\",\"database\"") {
		t.Fatalf("pages array must not reference removed broker/database sections")
	}
}

func TestLenseDashboardUsesGroupedLogRendererForErrors(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "IoTUI.renderGroupedLogs('errors'") {
		t.Fatalf("dashboard latest errors must reuse grouped log renderer")
	}
}
