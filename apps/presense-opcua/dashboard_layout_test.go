package main

import (
	"os"
	"strings"
	"testing"
)

func TestPresenseDashboardUsesSharedLenseLikeLayout(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		`overview-grid`,
		`health-layout`,
		`Health distribution`,
		`Health details`,
		`Connection graph`,
		`function renderPresenseGraph()`,
		`function renderHealthDetails()`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("presense dashboard should contain %s", want)
		}
	}
}
