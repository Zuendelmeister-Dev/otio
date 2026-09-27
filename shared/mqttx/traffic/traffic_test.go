package traffic

import (
	"strings"
	"testing"
	"time"
)

func TestRollingWindowAndBoundedPreview(t *testing.T) {
	var c Counter
	now := time.Unix(1800000000, 0)
	c.record(now.Add(-300*time.Second), "expired", []byte("old"))
	c.record(now.Add(-299*time.Second), "first", []byte("value"))
	c.record(now, "last", []byte(strings.Repeat("x", 4000)))
	s := c.snapshot(now)
	if s.Count != 2 || s.AvgPerMinute != .4 {
		t.Fatalf("window: %+v", s)
	}
	if !s.Last.Truncated || len(s.Last.Payload) != 2048 {
		t.Fatal("unbounded preview")
	}
	s.Last.Topic = "mutation"
	if c.snapshot(now).Last.Topic != "last" {
		t.Fatal("snapshot mutated counter")
	}
	s = c.snapshot(now.Add(301 * time.Second))
	if s.Count != 0 || s.AvgPerMinute != 0 || s.Last.Topic != "last" {
		t.Fatal("idle window must be zero and keep last message")
	}
	var restarted Counter
	if s = restarted.snapshot(now); s.Count != 0 || s.Last != nil || !s.Available {
		t.Fatal("restart must start empty")
	}
}

func TestRecentBoundedIndependentNewestFirst(t *testing.T) {
	var c Counter
	if len(c.Recent()) != 0 {
		t.Fatal("new counter not empty")
	}
	for i := 0; i < 70; i++ {
		c.record(time.Unix(int64(1000+i), 0), "topic", []byte("value"))
	}
	items := c.Recent()
	if len(items) != 50 || items[0].Timestamp.Unix() != 1069 || items[49].Timestamp.Unix() != 1020 {
		t.Fatalf("ring order: %+v", items)
	}
	items[0].Payload = "changed"
	if c.Recent()[0].Payload != "value" {
		t.Fatal("buffer leaked")
	}
}
