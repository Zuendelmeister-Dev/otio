package main

import (
	"os"
	"strings"
	"testing"
)

func TestAgentsListLinksStayInAgentsArea(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	for _, want := range []string{
		"link:'#agents/'+s.id,hint:'Protocol collector module'",
		"link:'#dashboard',hint:'Analytics, historian and central view'",
		"link:'#agents/'+x.id,hint:'Outbound forwarding module'",
		"link:'#agents/mqtt',hint:'Internal MQTT broker'",
		"link:'#agents/postgres',hint:'Historian database'",
		"async function loadAgent(id)",
		"findVirtualAgent(id)",
		"agentDetailBreadcrumbs",
		"<a href=\"#agents\">Agents</a>",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("agent navigation should contain %s", want)
		}
	}

	for _, notWant := range []string{
		"link:'#configurations/'+s.id,hint:'Protocol collector module'",
		"link:'#configurations/iot-lense',hint:'Analytics, historian and central view'",
		"link:'#configurations/'+x.id,hint:'Outbound forwarding module'",
	} {
		if strings.Contains(content, notWant) {
			t.Fatalf("agent navigation must not route list cards into configuration pages: %s", notWant)
		}
	}
}
