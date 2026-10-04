package app

import (
	"context"
	"testing"
	"time"

	"iot-lense-sense/sense/internal/protocols"
)

func TestSourceReadModeDefaultsToPoll(t *testing.T) {
	if got := sourceReadMode(SourceConfig{}); got != "poll" {
		t.Fatalf("read mode = %s, want poll", got)
	}
}

type repeatedSubscription struct{ cancel context.CancelFunc }

func (s repeatedSubscription) Subscribe(ctx context.Context, endpoint protocols.SourceEndpoint, metrics []protocols.MetricAddress, interval int, handler func(protocols.MetricValue)) error {
	for i := 0; i < 1000; i++ {
		handler(protocols.MetricValue{Name: "state", Value: "running", Quality: protocols.QualityGood})
	}
	s.cancel()
	return nil
}

func TestSubscriptionHealthDoesNotAccumulateEveryUpdate(t *testing.T) {
	previousState, previousRegistry := state, sourceProtocols
	t.Cleanup(func() { state, sourceProtocols = previousState, previousRegistry })
	state = &AppState{Sources: map[string]SourceStatus{}, Metrics: map[string]map[string][]MetricPoint{}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := protocols.NewRegistry()
	registry.RegisterSubscriber("stream", repeatedSubscription{cancel: cancel})
	sourceProtocols = registry
	subscribeSource(ctx, Config{}, SourceConfig{AgentID: "source", Type: "stream", Metrics: []MetricConfig{{Name: "state", Scale: 1}}})
	status := snapshotSources()["source"]
	if !status.Healthy || status.Message != "Subscription update received for state" {
		t.Fatalf("subscription accumulated a stream-sized health message: healthy=%v message length=%d", status.Healthy, len(status.Message))
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
