// Package traffic maintains bounded, process-local message flow statistics.
package traffic

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Timestamp time.Time `json:"timestamp"`
	Topic     string    `json:"topic"`
	Payload   string    `json:"payload"`
	Truncated bool      `json:"truncated"`
}
type Snapshot struct {
	Count        int64     `json:"count"`
	AvgPerMinute float64   `json:"avgPerMinute"`
	Last         *Message  `json:"last,omitempty"`
	SampledAt    time.Time `json:"sampledAt"`
	Since        time.Time `json:"since"`
	Available    bool      `json:"available"`
	Scope        string    `json:"scope"`
}
type bucket struct {
	second int64
	count  int64
}
type Counter struct {
	mu         sync.Mutex
	buckets    [300]bucket
	last       *Message
	since      time.Time
	messages   [50]Message
	next, size int
}

func clip(s string, n int) string {
	r := []rune(strings.ToValidUTF8(s, "�"))
	if len(r) > n {
		return string(r[:n])
	}
	return string(r)
}
func (c *Counter) Record(topic string, payload []byte) { c.record(time.Now(), topic, payload) }
func (c *Counter) record(now time.Time, topic string, payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.since.IsZero() {
		c.since = now
	}
	sec := now.Unix()
	b := &c.buckets[sec%300]
	if b.second != sec {
		*b = bucket{second: sec}
	}
	b.count++
	truncated := len(payload) > 2048
	if truncated {
		payload = payload[:2048]
	}
	c.last = &Message{Timestamp: now.UTC(), Topic: clip(topic, 256), Payload: strings.ToValidUTF8(string(payload), "�"), Truncated: truncated}
	c.messages[c.next] = *c.last
	c.next = (c.next + 1) % len(c.messages)
	if c.size < len(c.messages) {
		c.size++
	}
}
func (c *Counter) Snapshot() Snapshot { return c.snapshot(time.Now()) }
func (c *Counter) snapshot(now time.Time) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.since.IsZero() {
		c.since = now
	}
	s := Snapshot{SampledAt: now.UTC(), Since: c.since.UTC(), Available: true}
	for _, b := range c.buckets {
		if b.second > now.Unix()-300 && b.second <= now.Unix() {
			s.Count += b.count
		}
	}
	s.AvgPerMinute = float64(s.Count) / 5
	if c.last != nil {
		m := *c.last
		s.Last = &m
	}
	return s
}

// Recorder endpoints are intentionally same-origin; Lense aggregates configured instances.
func (c *Counter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(c.Snapshot())
}

// Recent returns an independent, newest-first bounded view.
func (c *Counter) Recent() []Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	items := make([]Message, c.size)
	for i := range items {
		items[i] = c.messages[(c.next-1-i+len(c.messages))%len(c.messages)]
	}
	return items
}
func (c *Counter) ServeMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(c.Recent())
}
