package main

import (
	"os"
	"strings"
	"testing"
)

func TestLenseSubNavigationIsCollapsible(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		`data-subnav-toggle="agents-subnav"`,
		`data-subnav-toggle="config-subnav"`,
		`.subnav{margin:0 0 6px 14px;display:none}`,
		`.subnav.open{display:block}`,
		`function syncSubnavState()`,
		`function ensureSubnavOpenForRoute(page)`,
		`event.preventDefault()`,
		`subnavOpenState`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("collapsible subnavigation should contain %s", want)
		}
	}
}
