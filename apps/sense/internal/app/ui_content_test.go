package app

import (
	"os"
	"strings"
	"testing"
)

func readUI(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestSenseNavigationMatchesLenseStructure(t *testing.T) {
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{`#dashboard`, `#analytics`, `#components`, `#config`, `#logs`, `#namespace`} {
		if !strings.Contains(content, want) {
			t.Fatalf("Sense navigation must contain %s", want)
		}
	}
	for _, removed := range []string{`#status`, `#metrics`, `page-broker`} {
		if strings.Contains(content, removed) {
			t.Fatalf("Sense navigation/page still references old route %s", removed)
		}
	}
}

func TestSenseDashboardContainsHealthAndSources(t *testing.T) {
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{`id="senseHealth"`, `id="senseHealthDetails"`, `id="sensePie"`, `id="sourcesConnected"`, `id="connectionGraph"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("Sense dashboard must contain %s", want)
		}
	}
}

func TestSenseComponentsContainBrokerOverview(t *testing.T) {
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{`id="senseComponentsOverview"`, `renderBrokerDetails`, `id="brokerDetails"`, `id="senseBrokerChart"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("Sense components must contain %s", want)
		}
	}
	if strings.Contains(content, `alert('Broker publishes`) {
		t.Fatalf("broker click must open component details instead of showing a message-count alert")
	}
}

func TestSenseConfigurationPageContainsGuardedWorkflow(t *testing.T) {
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{`id="page-config"`, `Validate changes`, `Apply proposal`, `id="configEditor"`, `id="history"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("Sense configuration page must contain %s", want)
		}
	}
}

func TestSenseUsesSharedUiHelpers(t *testing.T) {
	raw, err := os.ReadFile("../../static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{"IoTStandardChart.fillSelect", "IoTUI.renderGroupedLogs", "IoTUI.renderConnectionOverview", "IoTUI.renderOverviewCards"} {
		if !strings.Contains(content, want) {
			t.Fatalf("Sense must reuse shared helper %s", want)
		}
	}
}

func TestSenseConfigurationShowsApplyFeedbackToast(t *testing.T) {
	content := readUI(t)
	required := []string{
		"function showToast(",
		"Configuration applied",
		"Snapshot history was refreshed",
		"rollback-btn",
	}
	for _, item := range required {
		if !strings.Contains(content, item) {
			t.Fatalf("sense UI should contain %q", item)
		}
	}
}
