package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWatchdogDetectsHangAfterActivationAndAllowsRecovery(t *testing.T) {
	start := time.Unix(1000, 0)
	for _, tc := range []struct {
		name        string
		health      []bool
		seconds     []int
		wantFailure bool
	}{
		{"persistent hang", []bool{true, false, false}, []int{1, 2, 17}, true},
		{"transient failure", []bool{true, false, true, false, true}, []int{1, 2, 10, 17, 20}, false},
		{"startup failure", []bool{false, false}, []int{1, 120}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ticks := make(chan time.Time, len(tc.seconds))
			for _, second := range tc.seconds {
				ticks <- start.Add(time.Duration(second) * time.Second)
			}
			close(ticks)
			probe, activations, failures := 0, 0, 0
			supervise(context.Background(), ticks, start, watchdogConfig{2 * time.Minute, 15 * time.Second}, func() bool {
				value := tc.health[probe]
				probe++
				return value
			}, func(context.Context) error { activations++; return nil }, func() { failures++ })
			if (failures == 1) != tc.wantFailure {
				t.Fatalf("failures=%d", failures)
			}
			if tc.health[0] && activations != 1 {
				t.Fatalf("activated %d times", activations)
			}
		})
	}
}

func TestWatchdogBoundsFailedEndpointActivation(t *testing.T) {
	start := time.Unix(1000, 0)
	ticks := make(chan time.Time, 2)
	ticks <- start.Add(time.Second)
	ticks <- start.Add(2 * time.Minute)
	close(ticks)
	failures := 0
	supervise(context.Background(), ticks, start, watchdogConfig{2 * time.Minute, 15 * time.Second}, func() bool { return true }, func(context.Context) error { return errors.New("Kubernetes API unavailable") }, func() { failures++ })
	if failures != 1 {
		t.Fatalf("failed activation did not expire: %d", failures)
	}
}

func TestWatchdogCancellationDoesNotReactivate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	supervise(ctx, make(chan time.Time), time.Now(), watchdogConfig{}, func() bool { t.Fatal("probe after cancellation"); return false }, func(context.Context) error { t.Fatal("activation after cancellation"); return nil }, func() { t.Fatal("failure after cancellation") })
}

func TestWatchdogRejectsInvalidDurations(t *testing.T) {
	for _, raw := range []string{"nonsense", "0s", "-1s", "100ms"} {
		t.Setenv("HA_FAILURE_TIMEOUT", raw)
		if _, err := loadWatchdogConfig(); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}
