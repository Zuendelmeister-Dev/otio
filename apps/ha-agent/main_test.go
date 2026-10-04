package main

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestWatchdogTerminatesRunningChildWhenRequestsHang(t *testing.T) {
	t.Setenv("HA_TEST_CHILD", "1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &child{}
	defer c.stop()
	if err := c.start(ctx, []string{os.Args[0], "-test.run=^TestHelperProcess$"}, func() {}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	client := &http.Client{Timeout: 25 * time.Millisecond}
	start := time.Unix(1000, 0)
	ticks := make(chan time.Time, 3)
	for _, second := range []int{1, 2, 18} {
		ticks <- start.Add(time.Duration(second) * time.Second)
	}
	close(ticks)
	probes, activated, failed := 0, false, false
	supervise(ctx, ticks, start, watchdogConfig{time.Minute, 15 * time.Second}, func() bool {
		probes++
		if probes == 1 {
			return c.running()
		}
		response, err := client.Get(server.URL)
		if response != nil {
			response.Body.Close()
		}
		return err == nil
	}, func(context.Context) error { activated = true; return nil }, func() { failed = true; c.stop() })
	if !activated || !failed || c.running() {
		t.Fatal("unresponsive running application retained leadership")
	}
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
