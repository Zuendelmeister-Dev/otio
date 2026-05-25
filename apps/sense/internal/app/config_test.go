package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{
		Broker:         BrokerConfig{Host: "mqtt", Port: 1883, ClientID: "sense", TopicPrefix: "iot-lense"},
		PollIntervalMS: 1000,
		Sources: []SourceConfig{{
			AgentID: "machine-01",
			Type:    "modbus-tcp",
			Host:    "presense-modbus-01",
			Port:    5020,
			UnitID:  1,
			Metrics: []MetricConfig{{Name: "temperature", Register: 0, Scale: 0.1, Unit: "°C", Type: "gauge"}},
		}},
	}
}

func TestNormalizeJSON(t *testing.T) {
	got, err := normalizeJSON(`{"b":2,"a":1}`)
	if err != nil {
		t.Fatalf("normalizeJSON returned error: %v", err)
	}
	if !strings.Contains(got, "\n") || !strings.Contains(got, `"a": 1`) {
		t.Fatalf("normalizeJSON did not pretty-print as expected: %s", got)
	}
}

func TestValidateConfigValid(t *testing.T) {
	if problems := validateConfig(validConfig()); len(problems) != 0 {
		t.Fatalf("expected no validation problems, got %#v", problems)
	}
}

func TestValidateConfigProblems(t *testing.T) {
	cfg := validConfig()
	cfg.Broker.Host = ""
	cfg.PollIntervalMS = 50
	cfg.Sources[0].Metrics[0].Scale = 0
	problems := validateConfig(cfg)
	joined := strings.Join(problems, ";")
	for _, expected := range []string{"broker.host is required", "pollIntervalMs must be at least 100", "scale must not be 0"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in problems %#v", expected, problems)
		}
	}
}

func TestParseAndValidateRejectsUnknownFields(t *testing.T) {
	_, problems := parseAndValidate(`{"unknown":true}`)
	if len(problems) == 0 || !strings.Contains(problems[0], "unknown field") {
		t.Fatalf("expected unknown field problem, got %#v", problems)
	}
}

func TestMakeDiff(t *testing.T) {
	diff := makeDiff("a\nb", "a\nc")
	if !strings.Contains(diff, "- b") || !strings.Contains(diff, "+ c") {
		t.Fatalf("unexpected diff: %s", diff)
	}
}

func TestHistoryPruningAndListing(t *testing.T) {
	dir := t.TempDir()
	state.Lock()
	state.HistoryDir = dir
	state.Unlock()
	for i := 0; i < 7; i++ {
		if err := os.WriteFile(filepath.Join(dir, "20260101T00000"+string(rune('0'+i))+".json"), []byte(`{"n":`+string(rune('0'+i))+`}`), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := pruneHistory(dir, 5); err != nil {
		t.Fatal(err)
	}
	items := listHistory()
	if len(items) != 5 {
		t.Fatalf("len(history) = %d, want 5", len(items))
	}
}

func TestPruneHistoryDoesNotPanicWithFewFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "20260101T000000.000000000Z.json"), []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pruneHistory(dir, 5); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfigAcceptsOPCUASubscriptionMode(t *testing.T) {
	config := Config{
		Broker:         BrokerConfig{Host: "mqtt", Port: 1883, ClientID: "sense", TopicPrefix: "otio"},
		PollIntervalMS: 1000,
		Sources: []SourceConfig{{
			AgentID:                "opcua-machine-01",
			Type:                   "opcua",
			Host:                   "presense-opcua-01",
			Port:                   4840,
			ReadMode:               "subscription",
			SubscriptionIntervalMS: 1000,
			Metrics:                []MetricConfig{{Name: "temperature", NodeID: "ns=2;s=Machine.Temperature", Scale: 1, Type: "gauge"}},
		}},
	}
	if problems := validateConfig(config); len(problems) != 0 {
		t.Fatalf("problems = %v, want none", problems)
	}
}

func TestValidateConfigRejectsModbusSubscriptionMode(t *testing.T) {
	config := Config{
		Broker:         BrokerConfig{Host: "mqtt", Port: 1883, ClientID: "sense", TopicPrefix: "otio"},
		PollIntervalMS: 1000,
		Sources: []SourceConfig{{
			AgentID:  "modbus-machine-01",
			Type:     "modbus-tcp",
			Host:     "machine",
			Port:     502,
			UnitID:   1,
			ReadMode: "subscription",
			Metrics:  []MetricConfig{{Name: "temperature", Register: 0, Scale: 0.1, Type: "gauge"}},
		}},
	}
	problems := validateConfig(config)
	joined := strings.Join(problems, "; ")
	if !strings.Contains(joined, "subscription is currently supported for opcua only") {
		t.Fatalf("problems = %v, want modbus subscription rejection", problems)
	}
}
