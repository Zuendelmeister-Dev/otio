package main

import (
	"os"
	"strings"
	"testing"
)

func TestLenseUIUsesComponentStatusForGraph(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"function componentStatusMap()",
		"componentOnline(sense.id, false)",
		"const dispenseOk=componentOnline(group.dispense.id,false)",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing component status UI marker %s", want)
		}
	}
}
