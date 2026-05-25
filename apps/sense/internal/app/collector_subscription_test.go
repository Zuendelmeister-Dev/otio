package app

import (
	"testing"
	"time"
)

func TestSourceReadModeDefaultsToPoll(t *testing.T) {
	if got := sourceReadMode(SourceConfig{}); got != "poll" {
		t.Fatalf("read mode = %s, want poll", got)
	}
}

func TestSourceSubscriptionIntervalUsesLowerBound(t *testing.T) {
	config := Config{PollIntervalMS: 100}
	source := SourceConfig{SubscriptionIntervalMS: 100}
	if got := sourceSubscriptionInterval(source, config); got != 250*time.Millisecond {
		t.Fatalf("interval = %s, want 250ms", got)
	}
}

func TestScaleOPCUAValueKeepsStrings(t *testing.T) {
	metric := MetricConfig{Name: "state", Scale: 10}
	if got := scaleOPCUAValue(metric, "running"); got != "running" {
		t.Fatalf("value = %v, want running", got)
	}
}
