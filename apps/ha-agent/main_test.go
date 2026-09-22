package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("HA_TEST_CHILD") != "1" {
		return
	}
	time.Sleep(time.Minute)
	os.Exit(0)
}

func TestStopBeforeLeadershipPreventsLateStart(t *testing.T) {
	c := &child{}
	c.stop()
	if err := c.start(context.Background(), []string{"does-not-exist"}, func() {}); err != context.Canceled {
		t.Fatalf("late start: %v", err)
	}
}

func TestLeaseCancellationTerminatesApplication(t *testing.T) {
	t.Setenv("HA_TEST_CHILD", "1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exited := make(chan struct{})
	c := &child{}
	defer c.stop()
	if err := c.start(ctx, []string{os.Args[0], "-test.run=^TestHelperProcess$"}, func() { close(exited) }); err != nil {
		t.Fatal(err)
	}
	if !c.running() {
		t.Fatal("application was not started")
	}
	cancel()
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("application survived lease cancellation")
	}
	if c.running() {
		t.Fatal("exited application remains ready")
	}
	c.stop()
	c.stop()
}
