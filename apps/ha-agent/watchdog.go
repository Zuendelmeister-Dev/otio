package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

type watchdogConfig struct {
	startupTimeout time.Duration
	failureTimeout time.Duration
}

func loadWatchdogConfig() (watchdogConfig, error) {
	cfg := watchdogConfig{startupTimeout: 2 * time.Minute, failureTimeout: 15 * time.Second}
	for _, setting := range []struct {
		name   string
		target *time.Duration
	}{{"HA_STARTUP_TIMEOUT", &cfg.startupTimeout}, {"HA_FAILURE_TIMEOUT", &cfg.failureTimeout}} {
		if raw := os.Getenv(setting.name); raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil || value < time.Second {
				return cfg, fmt.Errorf("%s must be a duration of at least 1s", setting.name)
			}
			*setting.target = value
		}
	}
	return cfg, nil
}

// Keep supervising after activation: a live process may stop answering requests.
// A transient failed probe does not force a failover, and startup has its own budget.
func supervise(ctx context.Context, ticks <-chan time.Time, started time.Time, cfg watchdogConfig, healthy func() bool, activate func(context.Context) error, fail func()) {
	active := false
	var failedSince time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case now, open := <-ticks:
			if !open || ctx.Err() != nil {
				return
			}
			if healthy() {
				failedSince = time.Time{}
				if active {
					continue
				}
				if err := activate(ctx); err == nil {
					active = true
					continue
				} else {
					log.Printf("activate endpoint: %v", err)
				}
			} else if active && failedSince.IsZero() {
				failedSince = now
			}
			if (!active && now.Sub(started) >= cfg.startupTimeout) || (active && !failedSince.IsZero() && now.Sub(failedSince) >= cfg.failureTimeout) {
				log.Print("application health deadline exceeded; stopping leader for failover")
				fail()
				return
			}
		}
	}
}
