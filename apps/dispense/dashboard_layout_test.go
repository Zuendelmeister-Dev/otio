package main

import (
	"os"
	"strings"
	"testing"
)

func TestDispenseDashboardUsesSharedLenseLikeLayout(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		`id="dashboardCards"`,
		`id="dispenseHealthPie"`,
		`id="dispenseHealthDetails"`,
		`.overview-grid{display:grid`,
		`.health-layout{display:grid`,
		`IoTUI.renderOverviewCards('dashboardCards'`,
		`IoTUI.renderStatusDetails('dispenseHealthDetails'`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("dispense dashboard should contain %s", want)
		}
	}
}
