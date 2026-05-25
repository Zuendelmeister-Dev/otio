package main

import (
	"os"
	"strings"
	"testing"
)

func TestAgentSubNavigationExistsAndRoutesToGroups(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	for _, want := range []string{
		"agents-subnav",
		`href="#agents/presense"`,
		`href="#agents/sense"`,
		`href="#dashboard" class="lense" data-lense-self-target="agents"`,
		`href="#agents/dispense"`,
		`href="#agents/system"`,
		"function isAgentGroupSlug",
		"function agentGroupTitle",
		"loadAgents(isAgentGroupSlug(id)?id:null)",
		"currentAgentGroupSlug=groupSlug",
		"visibleGroups=groupSlug?groups.filter",
		"agentListBreadcrumbs",
		"agentListTitle",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("agent subgroup navigation should contain %s", want)
		}
	}
}

func TestAgentCardsUseOnlyGroupColorRails(t *testing.T) {
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
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("agent group color rail should contain %s", want)
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
